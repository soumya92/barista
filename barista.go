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
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/soumya92/barista/bar"
)

// i3Output is sent to i3bar.
type i3Output []map[string]interface{}

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
	Name       string
	LastOutput i3Output
	// Keep track of the paused/resumed state of the module.
	// Using a channel here allows concurrent pause/resume across
	// modules while guaranteeing ordering.
	paused   chan bool
	pausable bar.Pausable
}

// output converts the module's output to i3Output by adding the name (position),
// sets the module's last output to the converted i3Output, and signals the bar
// to update its output.
func (m *i3Module) output(ch chan<- interface{}) {
	for o := range m.Stream() {
		var i3out i3Output
		if o != nil {
			for _, segment := range o.Segments() {
				segmentOut := i3map(segment)
				segmentOut["name"] = m.Name
				i3out = append(i3out, segmentOut)
			}
		}
		m.LastOutput = i3out
		ch <- nil
	}
}

// loopPauseResume loops over values on the resumed channel and calls
// pause or resume on the wrapped module as appropriate.
func (m *i3Module) loopPauseResume() {
	for paused := range m.paused {
		if paused {
			m.pausable.Pause()
		} else {
			m.pausable.Resume()
		}
	}
}

// pause enqueues a pause call on the wrapped module
// if the wrapped module supports being paused.
func (m *i3Module) pause() {
	if m.paused != nil {
		m.paused <- true
	}
}

// resume enqueues a resume call on the wrapped module
// if the wrapped module supports being paused.
func (m *i3Module) resume() {
	if m.paused != nil {
		m.paused <- false
	}
}

// i3Bar is the "bar" instance that handles events and streams output.
type i3Bar struct {
	// The list of modules that make up this bar.
	i3Modules []*i3Module
	// The channel that receives a signal on module updates.
	update chan interface{}
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
}

var instance *i3Bar
var instanceInit sync.Once

func construct() {
	instanceInit.Do(func() {
		instance = &i3Bar{
			update: make(chan interface{}),
			events: make(chan i3Event),
			reader: os.Stdin,
			writer: os.Stdout,
		}
	})
}

// Add adds a module to the bar.
func Add(modules ...bar.Module) {
	construct()
	for _, m := range modules {
		instance.addModule(m)
	}
}

// SuppressSignals instructs the bar to skip the pause/resume signal handling.
// Must be called before Run.
func SuppressSignals(suppressSignals bool) {
	construct()
	if instance.started {
		panic("Cannot change signal handling after .Run()")
	}
	instance.suppressSignals = suppressSignals
}

// SetIo sets the input/output streams for the bar.
func SetIo(reader io.Reader, writer io.Writer) {
	construct()
	instance.reader = reader
	instance.writer = writer
}

// Run sets up all the streams and enters the main loop.
// If any modules are provided, they are added to the bar now.
// This allows both styles of bar construction:
// `bar.Add(a); bar.Add(b); bar.Run()`, and `bar.Run(a, b)`.
func Run(modules ...bar.Module) error {
	Add(modules...)
	// To allow ResetForTest to work, we need to avoid any references
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

	// Read events from the input stream, pipe them to the events channel.
	go b.readEvents()
	for _, m := range b.i3Modules {
		go m.output(b.update)
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

	// Infinite arrays on both sides.
	for {
		select {
		case _ = <-b.update:
			// The complete bar needs to printed on each update.
			if err := b.print(); err != nil {
				return err
			}
		case event := <-b.events:
			// Events are stripped of the name before being dispatched to the
			// correct module.
			if module, ok := b.get(event.Name); ok {
				// Check that the module actually supports click events.
				if clickable, ok := module.Module.(bar.Clickable); ok {
					// Goroutine to prevent click handlers from blocking the bar.
					go clickable.Click(event.Event)
				}
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
	i3Module := i3Module{
		Module: module,
		Name:   name,
	}
	if pauseable, ok := module.(bar.Pausable); ok {
		i3Module.paused = make(chan bool, 10)
		i3Module.pausable = pauseable
		go i3Module.loopPauseResume()
	}
	b.i3Modules = append(b.i3Modules, &i3Module)
}

// i3map serialises the attributes of the Segment in
// the format used by i3bar.
func i3map(s bar.Segment) map[string]interface{} {
	i3map := make(map[string]interface{})
	i3map["full_text"] = s.Text()
	if shortText, ok := s.GetShortText(); ok {
		i3map["short_text"] = shortText
	}
	if color, ok := s.GetColor(); ok {
		i3map["color"] = color
	}
	if background, ok := s.GetBackground(); ok {
		i3map["background"] = background
	}
	if border, ok := s.GetBorder(); ok {
		i3map["border"] = border
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
	// LastOutput property of each module will represent the current state.
	var outputs []map[string]interface{}
	for _, m := range b.i3Modules {
		for _, segment := range m.LastOutput {
			outputs = append(outputs, segment)
		}
	}
	if err := b.encoder.Encode(outputs); err != nil {
		return err
	}
	_, err := io.WriteString(b.writer, ",\n")
	return err
}

// get finds the module that corresponds to the given "name" from i3.
func (b *i3Bar) get(name string) (*i3Module, bool) {
	index, err := strconv.Atoi(name)
	if err != nil {
		return nil, false
	}
	if index < 0 || len(b.i3Modules) <= index {
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
	for _, m := range b.i3Modules {
		m.pause()
	}
}

// resume instructs all pausable modules to continue processing.
func (b *i3Bar) resume() {
	for _, m := range b.i3Modules {
		m.resume()
	}
}

// resetForTest resets the bar for testing purposes.
func resetForTest() {
	instance = nil
	instanceInit = sync.Once{}
}
