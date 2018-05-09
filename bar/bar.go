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
package bar

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

// Color represents a color string that will be handled by i3.
type Color string

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
	// We should support both chaining (e.g. segment.Urgent(true).Color(red))
	// and sequential calls (e.g. segment.Urgent(true); segment.Color(red);).
	// To do so, Segment needs to be mutable in-place, but making Segment
	// a reference type will disallow `return TextSegment("bad").Color(red)'.
	// To work around this, we wrap a reference type that holds all the data,
	// and have each method act on the inner field.
	*data
}

type data struct {
	// A bitmask of attributes that are set. Needed because the go default
	// for some attributes behave differently from unset values when sent to
	// i3bar. (e.g. the default separatorWidth is not 0).
	attrSet    int
	identifier string

	text      string
	shortText string
	markup    string

	color      Color
	background Color
	border     Color

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
	Segments() []Segment
}

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
	Button    Button `json:"button"`
	SegmentID string `json:"instance"`
	X         int    `json:"relative_x,omitempty"`
	Y         int    `json:"relative_y,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	ScreenX   int    `json:"x,omitempty"`
	ScreenY   int    `json:"y,omitempty"`
}

// Module represents a single bar module. A bar is just a list of modules.
type Module interface {
	// Stream will be called when the bar is started. The bar will then use the returned
	// output channel to update the module output, and use the last received output to
	// refresh the display when needed. Each new item on this channel will immediately
	// update the module output.
	Stream() <-chan Output
}

// Clickable is an additional interface modules may implement if they handle click events.
type Clickable interface {
	// Click will be called by the bar when it receives a mouse event from i3 that is
	// meant for this module.
	Click(Event)
}

// Ticker represents anything that can 'tick', and provides two ways to wait for updates.
// Tick() can be used in select {...}, while Wait() can be used in a for {...} loop.
type Ticker interface {
	// Tick returns a channel that sends nil each time the ticker "ticks".
	Tick() <-chan interface{}

	// Wait blocks until the ticker ticks. This is basically <-Tick().
	Wait()
}

// Scheduler represents a potentially repeating trigger and
// provides an interface to modify the trigger schedule.
type Scheduler interface {
	// The ticker ticks based on the trigger schedule set below.
	Ticker

	// At sets the scheduler to trigger a specific time.
	// This will replace any pending triggers.
	At(time.Time) Scheduler

	// After sets the scheduler to trigger after a delay.
	// This will replace any pending triggers.
	After(time.Duration) Scheduler

	// Every sets the scheduler to trigger at an interval.
	// This will replace any pending triggers.
	Every(time.Duration) Scheduler

	// Stop cancels all further triggers for the scheduler.
	Stop()
}

// Notifier provides a simple interface to handle notifying waiting
// clients that at least one update has occurred.
type Notifier interface {
	// The ticker ticks whenever the Notifier is marked as updated.
	Ticker

	// Notify marks this notifier as updated. If a listener is currently
	// waiting on the Ticker, it will immediately tick, otherwise the
	// next listener to start listening will receive the tick.
	// If no listeners are waiting, calling notify again is a nop.
	Notify()
}
