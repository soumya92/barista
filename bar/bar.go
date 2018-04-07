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

// Markup defines the output type of a block, e.g. plain text or pango format.
type Markup string

const (
	// MarkupNone represents plain-text output. No formatting will be parsed.
	MarkupNone = Markup("none")
	// MarkupPango represents pango formatting. Not all features may be supported.
	// See https://developer.gnome.org/pango/stable/PangoMarkupFormat.html.
	MarkupPango = Markup("pango")
)

// Color represents a color string that will be handled by i3.
type Color string

/*
Segment is a single "block" of output that conforms to the i3bar protocol.
See https://i3wm.org/docs/i3bar-protocol.html#_blocks_in_detail for details.

Note: Name is not included because only the bar needs to know the name in
order to dispatch click events and maintain the output cache. Multiple outputs
can still use the instance key (which the bar does not modify) to map click
events to output segments.

Since many of i3's default values do not match the default values in go for
their types, Segment is just a map[string]interface{} with typed methods
for setting values to allow distinguishing between unset values and values
that happen to match go's defaults. (e.g. separator = false, MinWidth = 0).

See output.go for supported methods.
*/
type Segment map[string]interface{}

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

The Instance is passed through unchanged from the output segments, so
it can be used to filter events for a module with multiple output segments.
*/
type Event struct {
	Button   Button `json:"button"`
	X        int    `json:"x,omitempty"`
	Y        int    `json:"y,omitempty"`
	Instance string `json:"instance"`
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

// Pausable is an additional interface modules may implement if they support being "paused".
type Pausable interface {
	// Pause will be called by the bar when it receives a SIGSTOP, usually when it is no
	// longer visible. Modules should use this as a signal to suspend background processing.
	Pause()

	// Resume will be called by the bar when it receives a SIGCONT, usually when it becomes
	// visible again. Modules should use this as a trigger for resuming background processing,
	// as well as immediately updating their output (or triggering a process to do so).
	Resume()
}
