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

/*
Package reformat provides a module that "wraps" an existing module and transforms it's output.
This can be useful for adding extra formatting simple bar modules.

For example, a time module might use strftime-style format strings,
which don't allow for colours or borders. You can add those using reformat:

 t := localtime.New(...)
 r := reformat.New(t).Format(func(o bar.Output) bar.Output {
   return o.Background("red").Padding(20)
 })
*/
package reformat

import (
	"sync"

	"github.com/soumya92/barista/bar"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// FormatFunc takes the module's output and returns a modified version.
type FormatFunc func(bar.Output) bar.Output

// Original returns the original output unchanged.
func Original(in bar.Output) bar.Output {
	return in
}

// Hide replaces all outputs with nil, hiding them from the bar.
func Hide(in bar.Output) bar.Output {
	return nil
}

// SegmentFunc is a reformatting function at the segment level.
type SegmentFunc func(*bar.Segment) *bar.Segment

// EachSegment transforms each segment individually.
func EachSegment(f SegmentFunc) FormatFunc {
	return func(in bar.Output) bar.Output {
		if in == nil {
			return in
		}
		out := outputs.Group()
		for _, s := range in.Segments() {
			out.Append(f(s.Clone()))
		}
		return out
	}
}

// SkipErrors wraps a segment transformation function so that
// error segments pass through unchanged.
func SkipErrors(f SegmentFunc) SegmentFunc {
	return func(in *bar.Segment) *bar.Segment {
		if in.GetError() != nil {
			return in
		}
		return f(in)
	}
}

// Module stores the original module, the re-formatting function, and
// helpers required to allow dynamic re-formatting.
type Module struct {
	module     bar.Module
	sink       bar.Sink
	mu         sync.Mutex
	formatter  FormatFunc
	restarted  chan bool
	lastOutput bar.Output
	started    bool
	finished   bool
}

// New wraps an existing bar.Module, allowing the format to be changed
// before being sent to the bar.
func New(original bar.Module) *Module {
	m := &Module{
		module:    original,
		restarted: make(chan bool),
		formatter: Original,
	}
	l.Label(m, l.ID(original))
	return m
}

// Format sets the reformat function.
func (m *Module) Format(f FormatFunc) *Module {
	l.Fine("%s.Format(%s)", l.ID(m), l.ID(f))
	m.mu.Lock()
	defer m.mu.Unlock()
	if f == nil {
		f = Original
	}
	m.formatter = f
	if m.started {
		go m.sink.Output(m.lastOutput)
	}
	return m
}

// Stream sets up the output pipeline to filter outputs when hidden.
func (m *Module) Stream(s bar.Sink) {
	wSink := wrappedSink(m, s)
	m.mu.Lock()
	m.sink = wSink
	m.mu.Unlock()
	for {
		m.module.Stream(wSink)
		m.mu.Lock()
		m.finished = true
		m.started = false
		m.mu.Unlock()
		<-m.restarted
		m.mu.Lock()
		nonErrorOutput := outputs.Group()
		formatted := m.formatter(m.lastOutput)
		if formatted != nil {
			for _, o := range formatted.Segments() {
				if o.GetError() == nil {
					nonErrorOutput.Append(o)
				}
			}
		}
		s.Output(nonErrorOutput)
		m.mu.Unlock()
	}
}

// Click passes through the click event if supported by the wrapped module.
func (m *Module) Click(e bar.Event) {
	m.mu.Lock()
	if m.finished && isRestartableClick(e) {
		l.Log("%s restarted", l.ID(m))
		m.finished = false
		m.restarted <- true
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()
	if clickable, ok := m.module.(bar.Clickable); ok {
		clickable.Click(e)
	}
}

// isRestartableClick mimics the function in barista main.
// Modules that have finished can still be reformatted,
// so the wrapping module needs to keep running.
func isRestartableClick(e bar.Event) bool {
	return e.Button == bar.ButtonLeft ||
		e.Button == bar.ButtonRight ||
		e.Button == bar.ButtonMiddle
}

func wrappedSink(m *Module, s bar.Sink) bar.Sink {
	return func(o bar.Output) {
		m.mu.Lock()
		m.lastOutput = o
		f := m.formatter
		m.started = true
		m.mu.Unlock()
		s.Output(f(o))
	}
}
