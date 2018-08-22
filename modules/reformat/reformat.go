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
	"sync/atomic"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/core"
	l "github.com/soumya92/barista/logging"
)

// FormatFunc takes the module's output and returns a modified version.
type FormatFunc = func(bar.Segments) bar.Output

// Original returns the original output unchanged.
func Original(in bar.Segments) bar.Output {
	return in
}

// Hide replaces all outputs with nil, hiding them from the bar.
func Hide(in bar.Segments) bar.Output {
	return nil
}

// Texts reformats a module's output with just the text content as input.
func Texts(f func(string) string) func(bar.Segments) bar.Output {
	return EachSegment(func(in *bar.Segment) *bar.Segment {
		txt, isPango := in.Content()
		if isPango {
			return in
		}
		return in.Text(f(txt))
	})
}

// SegmentFunc is a reformatting function at the segment level.
type SegmentFunc func(*bar.Segment) *bar.Segment

// EachSegment transforms each segment individually.
func EachSegment(f SegmentFunc) FormatFunc {
	return func(in bar.Segments) bar.Output {
		var out bar.Segments
		for _, s := range in {
			out = append(out, f(s.Clone()))
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

// Module wraps a bar.Module with a re-formatting function.
type Module struct {
	wrapped   *core.Module
	formatter atomic.Value // of FormatFunc
}

// New wraps an existing bar.Module, allowing the format to be changed
// before being sent to the bar.
func New(original bar.Module) *Module {
	m := &Module{wrapped: core.NewModule(original)}
	m.formatter.Store(Original)
	l.Label(m, l.ID(original))
	return m
}

// Format sets the reformat function.
func (m *Module) Format(f FormatFunc) *Module {
	l.Fine("%s.Format(%s)", l.ID(m), l.ID(f))
	if f == nil {
		f = Original
	}
	m.formatter.Store(f)
	m.wrapped.Replay()
	return m
}

// Stream sets up the output pipeline to filter outputs when hidden.
func (m *Module) Stream(s bar.Sink) {
	m.wrapped.Stream(wrappedSink(m, s))
}

// Click passes through the click event if supported by the wrapped module.
func (m *Module) Click(e bar.Event) {
	m.wrapped.Click(e)
}

func wrappedSink(m *Module, s bar.Sink) core.Sink {
	return func(o bar.Segments) {
		formatter := m.formatter.Load().(FormatFunc)
		s.Output(formatter(o))
	}
}
