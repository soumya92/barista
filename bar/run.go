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
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

// identifier stores a Name and instance pair used when communicating with i3bar.
// This information is never exposed outside the bar.
type identifier struct {
	Name     string `json:"name"`
	Instance string `json:"instance"`
}

// i3Output is sent to i3bar. It wraps the bar output and adds name and instance.
type i3Output struct {
	*Output
	identifier
}

// i3Event instances are received from i3bar on stdin.
type i3Event struct {
	Event
	identifier
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
	Identifier identifier
	LastOutput i3Output
}

// output converts the module's output to i3output by adding the name and instance.
func (m i3Module) output(ch chan<- i3Output) {
	for o := range m.Stream() {
		ch <- i3Output{Output: o, identifier: m.Identifier}
	}
}

// I3Bar is a "bar" instance that handles events and streams output.
type I3Bar struct {
	// The list of modules that make up this bar.
	i3Modules []i3Module
	// The channel that aggregates all the module outputs.
	outputs chan i3Output
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
	// Use the type, e.g. "*tztime.module", as the name. This doesn't matter,
	// but i3 would like to have it.
	name := reflect.TypeOf(module).String()
	// Use the position of the module in the list as the "instance", so when i3bar
	// sends us events, we can use atoi(instance) to get the correct module.
	instance := strconv.Itoa(len(b.i3Modules))
	i3Module := i3Module{
		Module:     module,
		Identifier: identifier{name, instance},
	}
	b.i3Modules = append(b.i3Modules, i3Module)
}

// Run sets up all the streams and enters the main loop.
func (b *I3Bar) Run() error {

	// Write header.
	header := i3Header{
		Version:     1,
		ClickEvents: true,
		// Go doesn't allow us to handle the default SIGSTOP,
		// so we'll use SIGUSR1 and SIGUSR2 for pause/resume.
		StopSignal: int(syscall.SIGUSR1),
		ContSignal: int(syscall.SIGUSR2),
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

	// Mark the bar as started.
	b.started = true

	// Read events from the input stream, pipe them to the events channel.
	go b.readEvents()
	// Merge all module outputs to the outputs channel.
	for _, m := range b.i3Modules {
		go m.output(b.outputs)
	}

	// Set up signal handlers for USR1/2 to pause/resume supported modules.
	signalChan := make(chan os.Signal, 2)
	signal.Notify(signalChan, syscall.SIGUSR1, syscall.SIGUSR2)

	// Infinite arrays on both sides.
	for {
		select {
		case out := <-b.outputs:
			// Any output from a module updates its "last output",
			// but the complete bar needs to printed each time.
			if module, ok := b.get(out.Instance); ok {
				module.LastOutput = out
				if err := b.print(); err != nil {
					return err
				}
			}
		case event := <-b.events:
			// Events are stripped of name and instance before being dispatched to the
			// correct module.
			if module, ok := b.get(event.Instance); ok {
				// Check that the module actually supports click events.
				if clickable, ok := module.Module.(Clickable); ok {
					// Goroutine to prevent click handlers from blocking the bar.
					go clickable.Click(event.Event)
				}
			}
		case sig := <-signalChan:
			switch sig {
			case syscall.SIGUSR1:
				b.pause()
			case syscall.SIGUSR2:
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
		outputs: make(chan i3Output),
		events:  make(chan i3Event),
		reader:  reader,
		writer:  writer,
	}
}

// print outputs the entire bar, using the last output for each module.
func (b *I3Bar) print() error {
	// i3bar requires the entire bar to be printed at once, so we just take the
	// last cached value for each module and construct the current bar.
	// The bar will update any modules before calling this method, so the
	// LastOutput property of each module will represent the current state.
	var outputs []i3Output
	for _, m := range b.i3Modules {
		out := m.LastOutput
		if out.Output == nil {
			continue
		}
		outputs = append(outputs, out)
	}
	if err := b.encoder.Encode(outputs); err != nil {
		return err
	}
	_, err := io.WriteString(b.writer, ",\n")
	return err
}

// get finds the module that corresponds to the given "instance" from i3.
func (b *I3Bar) get(instance string) (*i3Module, bool) {
	index, err := strconv.Atoi(instance)
	if err != nil {
		return nil, false
	}
	if index < 0 || len(b.i3Modules) <= index {
		return nil, false
	}
	return &b.i3Modules[index], true
}

// readEvents parses the infinite stream of events received from i3.
func (b *I3Bar) readEvents() {
	// Buffered I/O to allow complete events to be read in at once.
	reader := bufio.NewReader(b.reader)
	// Consume opening '['
	if rune, _, err := reader.ReadRune(); err != nil || rune != '[' {
		return
	}
	var event i3Event
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
		if pausable, ok := m.Module.(Pausable); ok {
			go pausable.Pause()
		}
	}
}

// resume instructs all pausable modules to continue processing.
func (b *I3Bar) resume() {
	for _, m := range b.i3Modules {
		if pausable, ok := m.Module.(Pausable); ok {
			go pausable.Resume()
		}
	}
}
