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

package bar

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

// i3Output is sent to i3bar.
type i3Output []map[string]interface{}

// i3Event instances are received from i3bar on stdin.
type i3Event struct {
	Event
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
	Module
	Name       string
	LastOutput i3Output
	// Keep track of the paused/resumed state of the module.
	// Using a channel here allows concurrent pause/resume across
	// modules while guaranteeing ordering.
	paused   chan bool
	pausable Pausable
}

// output converts the module's output to i3Output by adding the name (position),
// sets the module's last output to the converted i3Output, and signals the bar
// to update its output.
func (m *i3Module) output(ch chan<- interface{}) {
	for o := range m.Stream() {
		var i3out i3Output
		if o != nil {
			for _, segment := range o.Segments() {
				segmentOut := segment.i3map()
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

// I3Bar is a "bar" instance that handles events and streams output.
type I3Bar struct {
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

// Add adds a module to a bar, and returns the bar for chaining.
func (b *I3Bar) Add(modules ...Module) *I3Bar {
	for _, m := range modules {
		b.addModule(m)
	}
	// Return the bar for chaining (e.g. bar.Add(x, y).Run())
	return b
}

// addModule adds a single module to the bar.
func (b *I3Bar) addModule(module Module) {
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
	if pauseable, ok := module.(Pausable); ok {
		i3Module.paused = make(chan bool, 10)
		i3Module.pausable = pauseable
		go i3Module.loopPauseResume()
	}
	b.i3Modules = append(b.i3Modules, &i3Module)
}

// SuppressSignals instructs the bar to skip the pause/resume signal handling.
// Must be called before Run.
func (b *I3Bar) SuppressSignals(suppressSignals bool) *I3Bar {
	if b.started {
		panic("Cannot change signal handling after .Run()")
	}
	b.suppressSignals = suppressSignals
	return b
}

// Run sets up all the streams and enters the main loop.
func (b *I3Bar) Run() error {
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
				if clickable, ok := module.Module.(Clickable); ok {
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

// Run with a list of modules just adds each module and runs the bar on stdout/stdin.
func Run(modules ...Module) error {
	return New().Add(modules...).Run()
}

// RunOnIo takes a list of modules and the input/output streams, and runs the bar.
func RunOnIo(reader io.Reader, writer io.Writer, modules ...Module) error {
	return NewOnIo(reader, writer).Add(modules...).Run()
}

// New constructs a new bar running on standard I/O.
func New() *I3Bar {
	return NewOnIo(os.Stdin, os.Stdout)
}

// NewOnIo constructs a new bar with an input and output stream, for maximum flexibility.
func NewOnIo(reader io.Reader, writer io.Writer) *I3Bar {
	return &I3Bar{
		update: make(chan interface{}),
		events: make(chan i3Event),
		reader: reader,
		writer: writer,
	}
}

// print outputs the entire bar, using the last output for each module.
func (b *I3Bar) print() error {
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
func (b *I3Bar) get(name string) (*i3Module, bool) {
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
func (b *I3Bar) readEvents() {
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
func (b *I3Bar) pause() {
	for _, m := range b.i3Modules {
		m.pause()
	}
}

// resume instructs all pausable modules to continue processing.
func (b *I3Bar) resume() {
	for _, m := range b.i3Modules {
		m.resume()
	}
}
