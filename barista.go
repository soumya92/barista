// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package barista

import (
	"encoding/json"
	"errors"
	"image/color"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/sys/unix"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/core"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/timing"
)

// i3Event instances are received from i3bar on stdin.
type i3Event struct {
	bar.Event
	Name string `json:"name"`
}

// i3Header is sent at the beginning of output.
type i3Header struct {
	Version     int  `json:"version"`
	StopSignal  int  `json:"stop_signal,omitempty"`
	ContSignal  int  `json:"cont_signal,omitempty"`
	ClickEvents bool `json:"click_events"`
}

// i3Bar is the "bar" instance that handles events and streams output.
type i3Bar struct {
	sync.Mutex
	// The list of modules that make up this bar.
	modules   []bar.Module
	moduleSet *core.ModuleSet
	// To allow error outputs to show the full error using i3-nagbar without
	// forcing the bar into compact mode on long errors, store the full error
	// text for the last set of segments, and send error IDs to i3bar.
	// When a click event comes back with an error ID, intercept right-clicks
	// to show the error string instead of dispatching the event to the module.
	errors map[string]error
	// The function to call when an error segment is right-clicked.
	errorHandler func(bar.ErrorEvent)
	// The channel that receives a signal on module updates.
	update chan struct{}
	// The channel that aggregates all events from i3.
	events chan i3Event
	// The Reader to read events from (e.g. stdin)
	reader io.Reader
	// The Writer to write bar output to (e.g. stdout)
	writer io.Writer
	// A json encoder set to write to the output stream.
	encoder *json.Encoder
	// Flipped when Run() is called, to prevent issues with modules
	// being added after the bar has been started.
	started bool
	// Suppress pause/resume signal handling to workaround potential
	// weirdness with signals.
	suppressSignals bool
	// Keeps track of whether the bar is currently paused, and
	// whether it needs to be refreshed on resume.
	paused          bool
	refreshOnResume bool
	// For testing, output the associated error in the json as well.
	// This allows output tester to accurately check for errors.
	includeErrorsInOutput bool
	// For testing, emits debug events based on the requested mask.
	debugChan chan<- debugEvent
	debugMask int
}

type debugEventKind int

const (
	dEvtPaused debugEventKind = 1 << iota
	dEvtResumed
	dEvtModuleStopped
)

// debugEvent is used for tests to synchronise on some events that
// are otherwise extremely hard to user.
type debugEvent struct {
	kind debugEventKind
	data string
}

var instance *i3Bar
var instanceInit sync.Once

func construct() {
	instanceInit.Do(func() {
		instance = &i3Bar{
			update: make(chan struct{}, 1),
			events: make(chan i3Event),
			reader: os.Stdin,
			writer: os.Stdout,
			// bar starts paused, will be resumed on Run().
			paused: true,
			// Default to i3-nagbar when right-clicking errors.
			errorHandler: DefaultErrorHandler,
		}
	})
}

// Add adds a module to the bar.
func Add(module bar.Module) {
	construct()
	instance.Lock()
	defer instance.Unlock()
	if instance.started {
		panic("Cannot add modules after .Run()")
	}
	instance.modules = append(instance.modules, module)
}

// SuppressSignals instructs the bar to skip the pause/resume signal handling.
// Must be called before Run.
func SuppressSignals(suppressSignals bool) {
	construct()
	instance.Lock()
	defer instance.Unlock()
	if instance.started {
		panic("Cannot change signal handling after .Run()")
	}
	instance.suppressSignals = suppressSignals
}

// SetErrorHandler sets the function to be called when an error segment
// is right clicked. This replaces the DefaultErrorHandler.
func SetErrorHandler(handler func(bar.ErrorEvent)) {
	construct()
	instance.Lock()
	defer instance.Unlock()
	instance.errorHandler = handler
}

// Run sets up all the streams and enters the main loop.
// If any modules are provided, they are added to the bar now.
// This allows both styles of bar construction:
// `bar.Add(a); bar.Add(b); bar.Run()`, and `bar.Run(a, b)`.
func Run(modules ...bar.Module) error {
	construct()
	// To allow TestMode to work, we need to avoid any references
	// to instance in the run loop.
	b := instance
	var signalChan chan os.Signal
	if !b.suppressSignals {
		// Set up signal handlers for USR1/2 to pause/resume supported modules.
		signalChan = make(chan os.Signal, 2)
		signal.Notify(signalChan, unix.SIGUSR1, unix.SIGUSR2)
	}

	b.modules = append(b.modules, modules...)
	b.moduleSet = core.NewModuleSet(b.modules)

	// Mark the bar as started.
	b.started = true
	l.Log("Bar started")

	go func(i <-chan int) {
		for range i {
			b.refresh()
		}
	}(b.moduleSet.Stream())

	errChan := make(chan error)
	// Read events from the input stream, pipe them to the events channel.
	go func(e chan<- error) {
		e <- b.readEvents()
	}(errChan)

	// Write header.
	header := i3Header{
		Version:     1,
		ClickEvents: true,
	}

	if !b.suppressSignals {
		// Go doesn't allow us to handle the default SIGSTOP,
		// so we'll use SIGUSR1 and SIGUSR2 for pause/resume.
		header.StopSignal = int(unix.SIGUSR1)
		header.ContSignal = int(unix.SIGUSR2)
	}
	// Set up the encoder for the output stream,
	// so that module outputs can be written directly.
	b.encoder = json.NewEncoder(b.writer)
	if err := b.encoder.Encode(&header); err != nil {
		return err
	}
	// Start the infinite array.
	if _, err := io.WriteString(b.writer, "["); err != nil {
		return err
	}

	// Bar starts paused, so resume it to get the initial output.
	b.resume()

	// Infinite arrays on both sides.
	for {
		select {
		case <-b.update:
			// The complete bar needs to printed on each update.
			if err := b.print(); err != nil {
				return err
			}
		case event := <-b.events:
			l.Fine("Clicked on '%s'", event.Name)
			moduleID, errorID := idFromName(event.Name)
			// If the clicked segment has an error, intercept right clicks
			// and show nagbar. Everything else is handled as normal.
			if errorID != "" && event.Button == bar.ButtonRight {
				if err, ok := b.errors[errorID]; ok {
					go b.errorHandler(bar.ErrorEvent{
						Error: err,
						Event: event.Event,
					})
				}
				continue
			}
			// Events are stripped of the name before being dispatched to the
			// correct module.
			b.click(moduleID, event.Event)
		case sig := <-signalChan:
			switch sig {
			case unix.SIGUSR1:
				b.pause()
			case unix.SIGUSR2:
				b.resume()
			}
		case err := <-errChan:
			return err
		}
	}
}

func idFromName(name string) (moduleID, errorID string) {
	parts := strings.Split(name, "/")
	assertLen := func(expected int) bool {
		if len(parts) != expected {
			l.Log("Unexpected name: %s", name)
			return false
		}
		return true
	}
	// name format: m/$mod or e/$err/$mod.
	switch parts[0] {
	case "m":
		if assertLen(2) {
			moduleID = parts[1]
		}
	case "e":
		if assertLen(3) {
			errorID = parts[1]
			moduleID = parts[2]
		}
	}
	return moduleID, errorID
}

// DefaultErrorHandler invokes i3-nagbar to show the full error message.
func DefaultErrorHandler(e bar.ErrorEvent) {
	exec.Command("i3-nagbar", "-m", e.Error.Error()).Run()
}

func colorString(c color.Color) string {
	cful, _ := colorful.MakeColor(c)
	return cful.Hex()
}

// i3map serialises the attributes of the Segment in
// the format used by i3bar.
func i3map(s *bar.Segment) map[string]interface{} {
	i3map := make(map[string]interface{})
	txt, pango := s.Content()
	i3map["full_text"] = txt
	if shortText, ok := s.GetShortText(); ok {
		i3map["short_text"] = shortText
	}
	if color, ok := s.GetColor(); ok {
		i3map["color"] = colorString(color)
	}
	if background, ok := s.GetBackground(); ok {
		i3map["background"] = colorString(background)
	}
	if border, ok := s.GetBorder(); ok {
		i3map["border"] = colorString(border)
	}
	if minWidth, ok := s.GetMinWidth(); ok {
		i3map["min_width"] = minWidth
	}
	if align, ok := s.GetAlignment(); ok {
		i3map["align"] = align
	}
	if id, ok := s.GetID(); ok {
		i3map["instance"] = id
	}
	if urgent, ok := s.IsUrgent(); ok {
		i3map["urgent"] = urgent
	}
	if separator, ok := s.HasSeparator(); ok {
		i3map["separator"] = separator
	}
	if padding, ok := s.GetPadding(); ok {
		i3map["separator_block_width"] = padding
	}
	if pango {
		i3map["markup"] = "pango"
	} else {
		i3map["markup"] = "none"
	}
	return i3map
}

// print outputs the entire bar, using the last output for each module.
func (b *i3Bar) print() error {
	// i3bar requires the entire bar to be printed at once, so we just take the
	// last cached value for each module and construct the current bar.
	// The bar will update any modules before calling this method, so the
	// lastOutput property of each module will represent the current state.
	b.errors = map[string]error{}
	output := make([]map[string]interface{}, 0)
	for idx, segments := range b.moduleSet.LastOutputs() {
		name := strconv.Itoa(idx)
		for _, segment := range segments {
			out := i3map(segment)
			if err := segment.GetError(); err != nil {
				errorID := strconv.Itoa(len(b.errors))
				b.errors[errorID] = err
				out["name"] = "e/" + errorID + "/" + name
				if b.includeErrorsInOutput {
					out["error"] = err.Error()
				}
			} else {
				out["name"] = "m/" + name
			}
			output = append(output, out)
		}
	}
	if err := b.encoder.Encode(output); err != nil {
		return err
	}
	_, err := io.WriteString(b.writer, ",\n")
	return err
}

// click sends the event to the module of the given "name".
func (b *i3Bar) click(name string, evt bar.Event) {
	index, err := strconv.Atoi(name)
	if err != nil {
		l.Log("Malformed module id '%s'", name)
		return
	}
	if index < 0 || len(b.modules) <= index {
		l.Log("Could not find module '%s'", name)
		return
	}
	go b.moduleSet.Click(index, evt)
}

// readEvents parses the infinite stream of events received from i3.
func (b *i3Bar) readEvents() error {
	decoder := json.NewDecoder(b.reader)
	// Consume opening '['
	_, err := decoder.Token()
	if err != nil {
		return err
	}
	for decoder.More() {
		var event i3Event
		err = decoder.Decode(&event)
		if err != nil {
			return err
		}
		b.events <- event
	}
	return errors.New("stdin exhausted")
}

// pause instructs all pausable modules to suspend processing.
func (b *i3Bar) pause() {
	l.Log("Bar paused")
	b.Lock()
	defer b.Unlock()
	if b.paused {
		return
	}
	b.paused = true
	timing.Pause()
	b.emitDebugEvent(dEvtPaused, "")
}

// resume instructs all pausable modules to continue processing.
func (b *i3Bar) resume() {
	l.Log("Bar resumed")
	b.Lock()
	defer b.Unlock()
	if !b.paused {
		return
	}
	b.paused = false
	timing.Resume()
	if b.refreshOnResume {
		b.refreshOnResume = false
		b.maybeUpdate()
	}
	b.emitDebugEvent(dEvtResumed, "")
}

// refresh requests an update of the bar's output.
func (b *i3Bar) refresh() {
	b.Lock()
	defer b.Unlock()
	// If paused, defer the refresh until the bar resumes.
	if b.paused {
		l.Fine("Refresh on next resume")
		b.refreshOnResume = true
		return
	}
	b.maybeUpdate()
}

// maybeUpdate signals the update channel unless already signalled.
func (b *i3Bar) maybeUpdate() {
	select {
	case b.update <- struct{}{}:
	default:
		// Since b.update has a buffer of 1, a failure to send to it
		// implies that an update is already queued. Since refresh
		// is only be called after individual modules' lastOutput is
		// set, when the previous update is consumed, each module will
		// already have the latest output.
	}
}

// emitDebugEvent emits a debug event if the channel is not nil
// and events of the kind have been requested.
func (b *i3Bar) emitDebugEvent(kind debugEventKind, data string) {
	if b.debugMask&int(kind) != 0 {
		b.debugChan <- debugEvent{kind, data}
	}
}

// TestMode creates a new instance of the bar for testing purposes,
// and runs it on the provided streams instead of stdin/stdout.
func TestMode(reader io.Reader, writer io.Writer) {
	instanceInit = sync.Once{}
	construct()
	instance.Lock()
	defer instance.Unlock()
	instance.reader = reader
	instance.writer = writer
	instance.includeErrorsInOutput = true
}
