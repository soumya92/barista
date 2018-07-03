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

// Errorf constructs a bar output that indicates an error,
// using the given format string and arguments.
func Errorf(format string, args ...interface{}) *bar.Segment {
	return Error(fmt.Errorf(format, args...))
}

// Error constructs a bar output that indicates an error.
func Error(e error) *bar.Segment {
	return bar.ErrorSegment(e)
}

// Textf constructs simple text output from a format string and arguments.
func Textf(format string, args ...interface{}) *bar.Segment {
	return Text(fmt.Sprintf(format, args...))
}

//Text constructs a simple text output from the given string.
func Text(text string) *bar.Segment {
	return bar.TextSegment(text)
}

// Pango constructs a pango bar segment from a list of pango Nodes and strings.
func Pango(things ...interface{}) *bar.Segment {
	nodes := []*pango.Node{}
	for _, thing := range things {
		switch t := thing.(type) {
		case *pango.Node:
			nodes = append(nodes, t)
		case string:
			nodes = append(nodes, pango.Text(t))
		default:
			nodes = append(nodes, pango.Textf("%v", t))
		}
	}
	return bar.PangoSegment(pango.New(nodes...).Pango())
}

// Group concatenates several outputs into a single SegmentGroup,
// to facilitate easier manipulation of output properties.
// For example, setting a colour or urgency for all segments together.
func Group(outputs ...bar.Output) *SegmentGroup {
	group := newSegmentGroup()
	for _, o := range outputs {
		if o != nil {
			group.Append(o.Segments()...)
		}
	}
	return group
}
