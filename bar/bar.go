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

// Package bar allows a user to create a go binary that follows the i3bar protocol.
package bar // import "barista.run/bar"

import "image/color"
import "time"

// TextAlignment defines the alignment of text within a block.
// Using TextAlignment rather than string opens up the possibility of i18n without
// requiring each module to know the current locale.
type TextAlignment string

const (
	// AlignStart aligns text to the start of the module, which is left for LTR languages.
	AlignStart = TextAlignment("left")
	// AlignCenter aligns text to the middle of the module.
	AlignCenter = TextAlignment("center")
	// AlignEnd aligns text to the end of the module, which is right for LTR languages.
	AlignEnd = TextAlignment("right")
)

/*
Segment is a single "block" of output that conforms to the i3bar protocol.
See https://i3wm.org/docs/i3bar-protocol.html#_blocks_in_detail for details.

Note: Name is not included because only the bar needs to know the name in
order to dispatch click events and maintain the output cache. Multiple segments
can still use the identifier to map click events to output segments.
The bar will map the unmodified identifier to i3bar's "instance", and set the
value from the clicked segment as the SegmentID of the generated event.

See segment.go for supported methods. All fields are unexported to make sure
that when setting a field, the attrSet mask is also updated.
*/
type Segment struct {
	// A bitmask of attributes that are set. Needed because the go default
	// for some attributes behave differently from unset values when sent to
	// i3bar. (e.g. the default separatorWidth is not 0).
	attrSet int
	onClick func(Event)

	text      string
	pango     bool
	shortText string
	err       error

	color      color.Color
	background color.Color
	border     color.Color

	// Minimum width can be specified as either a numeric pixel value
	// or a string placeholder value. The unexported field is interface{}
	// but there are two methods on Segment that set this, one for each type.
	minWidth interface{}

	align     TextAlignment
	urgent    bool
	separator bool
	padding   int
}

// sa* (Segment Attribute) consts are used as bitwise flags in attrSet
// to indicate which attributes are set (and so should be serialised).
const (
	saShortText int = 1 << iota
	saUrgent
	saSeparator
	saPadding
)

// Output is an interface for displaying objects on the bar.
type Output interface {
	Segments() []*Segment
}

// TimedOutput extends bar.Output with a hint that indicates the next time that
// the output segments will be different. This can be used, for example, to show
// elapsed duration since a fixed point in time.
type TimedOutput interface {
	Output
	NextRefresh() time.Time
}

// Segments implements Output for []*Segment.
type Segments []*Segment

// Button represents an X11 mouse button.
type Button int

const (
	// ButtonLeft is the left mouse button.
	ButtonLeft Button = 1
	// ButtonRight is the right mouse button.
	ButtonRight Button = 3
	// ButtonMiddle is the middle mouse button, sometimes the scroll-wheel.
	ButtonMiddle Button = 2
	// ButtonBack is the "back" button, usually on the side.
	ButtonBack Button = 8
	// ButtonForward is the "forward" button, usually next to the back button.
	ButtonForward Button = 9

	// ScrollUp on the mouse wheel.
	ScrollUp Button = 4
	// ScrollDown on the mouse wheel.
	ScrollDown Button = 5
	// ScrollLeft or tilt left on the mouse wheel.
	ScrollLeft Button = 6
	// ScrollRight or tilt right on the mouse wheel.
	ScrollRight Button = 7
)

/*
Event represents a mouse event meant for a single module.

Note: As before, name is not included because it's only required to determine
which module will handle an event from i3. Once the bar receives the event,
it provides only the information in this struct to individual modules.

The SegmentID is set to the Identifier of the output segment clicked, so
it can be used to filter events for a module with multiple output segments.

X, Y describe event co-ordinates relative to the output segment, and
Width, Height are set to the size of the output segment.

ScreenX, ScreenY are the event co-ordinates relative to the root window.
*/
type Event struct {
	Button  Button `json:"button"`
	X       int    `json:"relative_x,omitempty"`
	Y       int    `json:"relative_y,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	ScreenX int    `json:"x,omitempty"`
	ScreenY int    `json:"y,omitempty"`
}

/*
ErrorEvent represents a mouse event that triggered the error handler.
This is fired when an error segment is right clicked. The default handler
for ErrorEvents simply shows an i3-nagbar with the full error text.

Since the Event that triggered the error handler is also embedded,
error handlers have information about the position of the module and can
choose to display more contextual messages than a simple bar across the
entire screen.
*/
type ErrorEvent struct {
	Error error
	Event
}

// Sink represents a destination for module output.
type Sink func(Output)

// Module represents a single bar module. A bar is just a list of modules.
type Module interface {
	// Stream runs the main loop of a module, pushing updated outputs to
	// the provided Sink.
	// A module is considered active until Stream() returns, at which point
	// a click will restart the module by calling Stream() again.
	// The Sink passed to Stream is only valid for the one call to Stream;
	// subsequent calls may receive different instances.
	Stream(Sink)
}

// RefresherModule extends module with a Refresh() method that forces a refresh
// of the data being displayed (e.g. a fresh HTTP request or file read).
// core.Module will add middle-click to refresh for modules that implement it.
type RefresherModule interface {
	Module
	Refresh()
}
