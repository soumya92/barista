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

// Package outputs provides helper functions to construct bar.Outputs.
package outputs

import (
	"fmt"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/pango"
)

// TemplateFunc is a function that takes in a single argument constructs a
// bar output from it.
type TemplateFunc func(interface{}) bar.Output

// empty represents an empty output.
type empty struct{}

// Segments implements bar.Output for empty by returning an empty list.
func (e empty) Segments() []bar.Segment {
	return []bar.Segment{}
}

// Empty constructs an empty output, which will hide a module from the bar.
func Empty() bar.Output {
	return empty{}
}

// Errorf constructs a bar output that indicates an error,
// using the given format string and arguments.
func Errorf(format string, args ...interface{}) bar.Output {
	return Error(fmt.Errorf(format, args...))
}

// Error constructs a bar output that indicates an error.
func Error(e error) bar.Output {
	return bar.NewSegment(e.Error()).
		ShortText("Error").
		Urgent(true)
}

// Textf constructs simple text output from a format string and arguments.
func Textf(format string, args ...interface{}) bar.Segment {
	return Text(fmt.Sprintf(format, args...))
}

//Text constructs a simple text output from the given string.
func Text(text string) bar.Segment {
	return bar.NewSegment(text)
}

// PangoUnsafe constructs a bar output from existing pango markup.
// This function does not perform any escaping.
func PangoUnsafe(markup string) bar.Segment {
	return bar.NewSegment(markup).Markup(bar.MarkupPango)
}

// Pango constructs a bar output from a list of things.
func Pango(things ...interface{}) bar.Segment {
	// The extra span tag will be collapsed if no attributes were added.
	return PangoUnsafe(pango.Span(things...).Pango())
}

// Group merges several outputs into a single SegmentGroup, to facilitate
// easier manipulation of output properties (e.g. colour, urgency).
func Group(outputs ...bar.Output) bar.SegmentGroup {
	out := []bar.Segment{}
	for _, o := range outputs {
		out = append(out, o.Segments()...)
	}
	return bar.SegmentGroup(out)
}
