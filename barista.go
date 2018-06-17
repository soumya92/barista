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
	"bufio"
	"encoding/json"
	"image/color"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lucasb-eyer/go-colorful"
	"golang.org/x/sys/unix"

	"github.com/soumya92/barista/bar"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/timing"
)

type i3Segment map[string]interface{}

type i3SegmentWithError struct {
	i3Segment
	error
}

type i3Output []i3SegmentWithError

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

// i3Module wraps Module with extra information to help run i3bar.
type i3Module struct {
	bar.Module
	name       string
	lastOutput atomic.Value // of i3Output
	// If the Stream() channel is closed, the next left/middle/right click
	// event will restart the module. This provides an easy way for modules
	// to report errors: `out <- outputs.Error(...); close(out);`
	// (where out = the output channel of the module, returned from Stream).
	restartable atomic.Value // of bool
}

// i3Bar is the "bar" instance that handles events and streams output.
type i3Bar struct {
	sync.Mutex
	// The list of modules that make up this bar.
	i3Modules []*i3Module
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
}

// output converts the module's output to i3Output by adding the name (position),
// sets the module's last output to the converted i3Output, and signals the bar
// to update its output.
func (m *i3Module) output(b *i3Bar) {
	m.Stream(func(o bar.Output) {
		var i3out i3Output
		if o != nil {
			for _, segment := range o.Segments() {
				i3out = append(i3out, i3SegmentWithError{
					i3map(segment),
					segment.GetError(),
				})
			}
		}
		m.lastOutput.Store(i3out)
		l.Fine("New output from %s", l.ID(m.Module))
		b.refresh()
	})
	// If we got here, the module is finished, so we mark the module
	// as "restartable" and the next click event (Button1/2/3) will
	// call the output() method again.
	m.restartable.Store(true)
}

// handleRestart checks if the module needs to be restarted, and restarts
// the module if the event is a left, right, or middle click. It clears
// the restartable flag and calls Stream on the module to resume output
// to the bar. It also clears any error segments.
// Returns true if the event was swallowed (i.e. should not be dispatched)
// to the underlying module.
func (m *i3Module) handleRestart(bar *i3Bar, e bar.Event) bool {
	needsRestart, _ := m.restartable.Load().(bool)
	if !needsRestart {
		// Process event normally.
		return false
	}
	if !isRestartableClick(e) {
		// Swallow event, but do not restart the module.
		return true
	}
	// Restart the module.
	lastOut := m.lastOutput.Load().(i3Output)
	var newOut i3Output
	for _, segment := range lastOut {
		if segment.error == nil {
			newOut = append(newOut, segment)
		}
	}
	if len(lastOut) != len(newOut) {
		l.Fine("Module %s cleared %d error output(s)",
			l.ID(m.Module), len(lastOut)-len(newOut))
		m.lastOutput.Store(newOut)
		bar.refresh()
	}
	m.restartable.Store(false)
	go m.output(bar)
	l.Log("Module %s restarted", l.ID(m.Module))
	// Swallow the event.
	return true
}

// isRestartableClick checks whether an event is allowed to restart
// a pending module. Only left/middle/right clicks restart a module.
func isRestartableClick(e bar.Event) bool {
	return e.Button == bar.ButtonLeft ||
		e.Button == bar.ButtonRight ||
		e.Button == bar.ButtonMiddle
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
	instance.addModule(module)
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
	for _, m := range modules {
		instance.addModule(m)
	}
	// To allow TestMode to work, we need to avoid any references
	// to instance in the run loop.
	b := instance
	var signalChan chan os.Signal
	if !b.suppressSignals {
		// Set up signal handlers for USR1/2 to pause/resume supported modules.
		signalChan = make(chan os.Signal, 2)
		signal.Notify(signalChan, unix.SIGUSR1, unix.SIGUSR2)
	}

	// Mark the bar as started.
	b.started = true
	l.Log("Bar started")

	// Read events from the input stream, pipe them to the events channel.
	go b.readEvents()
	for _, m := range b.i3Modules {
		go m.output(b)
		l.Log("Module %s started", l.ID(m.Module))
	}

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
			module, ok := b.get(moduleID)
			if !ok {
				continue
			}
			l.Fine("Clicked on module %s", l.ID(module.Module))
			// If the module swallows the event to potentially restart,
			// do not dispatch it to the click handler.
			if module.handleRestart(b, event.Event) {
				continue
			}
			// Check that the module actually supports click events.
			if clickable, ok := module.Module.(bar.Clickable); ok {
				// Goroutine to prevent click handlers from blocking the bar.
				go clickable.Click(event.Event)
			}
		case sig := <-signalChan:
			switch sig {
			case unix.SIGUSR1:
				b.pause()
			case unix.SIGUSR2:
				b.resume()
			}
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

// addModule adds a single module to the bar.
func (b *i3Bar) addModule(module bar.Module) {
	// Panic if adding modules to an already running bar.
	// TODO: Support this in the future.
	if b.started {
		panic("Cannot add modules after .Run()")
	}
	// Use the position of the module in the list as the "name", so when i3bar
	// sends us events, we can use atoi(name) to get the correct module.
	name := strconv.Itoa(len(b.i3Modules))
	l.Log("Module '%s' -> %s", name, l.ID(module))
	i3Module := i3Module{
		Module: module,
		name:   name,
	}
	b.Lock()
	defer b.Unlock()
	b.i3Modules = append(b.i3Modules, &i3Module)
}

func colorString(c color.Color) string {
	cful, _ := colorful.MakeColor(c)
	return cful.Hex()
}

// i3map serialises the attributes of the Segment in
// the format used by i3bar.
func i3map(s *bar.Segment) map[string]interface{} {
	i3map := make(map[string]interface{})
	i3map["full_text"] = s.Text()
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
	if s.IsPango() {
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
	output := make([]i3Segment, 0)
	for _, m := range b.i3Modules {
		lastOut, ok := m.lastOutput.Load().(i3Output)
		if !ok {
			continue
		}
		for _, segment := range lastOut {
			out := segment.i3Segment
			if segment.error != nil {
				errorID := strconv.Itoa(len(b.errors))
				b.errors[errorID] = segment.error
				out["name"] = "e/" + errorID + "/" + m.name
				if b.includeErrorsInOutput {
					out["error"] = segment.Error()
				}
			} else {
				out["name"] = "m/" + m.name
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

// get finds the module that corresponds to the given "name" from i3.
func (b *i3Bar) get(name string) (*i3Module, bool) {
	index, err := strconv.Atoi(name)
	if err != nil {
		l.Log("Malformed module id '%s'", name)
		return nil, false
	}
	if index < 0 || len(b.i3Modules) <= index {
		l.Log("Could not find module '%s'", name)
		return nil, false
	}
	return b.i3Modules[index], true
}

// readEvents parses the infinite stream of events received from i3.
func (b *i3Bar) readEvents() {
	// Buffered I/O to allow complete events to be read in at once.
	reader := bufio.NewReader(b.reader)
	// Consume opening '['
	if rune, _, err := reader.ReadRune(); err != nil || rune != '[' {
		return
	}
	for {
		// While the 'proper' way to implement this infinite parser would be to keep
		// a state machine and hook into json parsing and stuff, we'll take a
		// shortcut since we know there are no nested objects. So all we have to do
		// is read until the first '}', decode it, consume the ',', and repeat.
		eventJSON, err := reader.ReadString('}')
		if err != nil {
			return
		}
		// The '}' is consumed by ReadString, but required by json Decoder.
		event := i3Event{}
		decoder := json.NewDecoder(strings.NewReader(eventJSON + "}"))
		err = decoder.Decode(&event)
		if err != nil {
			return
		}
		b.events <- event
		// Consume ','
		if _, err := reader.ReadString(','); err != nil {
			return
		}
	}
}

// pause instructs all pausable modules to suspend processing.
func (b *i3Bar) pause() {
	l.Log("Bar paused")
	b.Lock()
	defer b.Unlock()
	b.paused = true
	timing.Pause()
}

// resume instructs all pausable modules to continue processing.
func (b *i3Bar) resume() {
	l.Log("Bar resumed")
	b.Lock()
	defer b.Unlock()
	b.paused = false
	timing.Resume()
	if b.refreshOnResume {
		b.refreshOnResume = false
		b.maybeUpdate()
	}
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
