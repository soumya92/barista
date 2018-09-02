---
title: Bar Interfaces & Types
---

# Segment

A `bar.Segment` represents a single segment of output. It defines the text and markup style, as well
as formatting attributes like colours, separators, and widths. See the godoc for the full list.

It is used as a pointer, to allow both styles of construction: `new(Segment).Color(red).Width(100)`
and `segment = new(Segment); segment.Color(red); segment.Width(100)`. To create a changed version
of an existing segment without the changes being reflected in the original, use `Clone()`. This can
also be used to create "template" segments, which can then be used as `tpl.Clone().Text("foo")` to
create new segments that have all the same properties as the template.

# Output

A `bar.Output` represents something that can be expressed as a collection of `bar.Segments`. Because
this is an interface, many types to be used for output directly. For example, `*pango.Node`,
`*bar.Segment`, and `bar.Segments` all implement `bar.Output` and can be used wherever an output
is required.

# Sink

A `bar.Sink` represents a destination for `bar.Output`. The main barista instance will provide sinks
that output to stdout for consumption by i3bar, but other modules may provide their own sinks to
process values before sending them on (e.g. [reformat](/modules/reformat) or [group](/group)).

Sink also provides a convenience function to check errors: `Error(error) boolean`. If the error is
nil, this function simply returns false, but if the error is non-nil, it sends an error segment to
the sink and returns true. This allows concise error handling in modules:

```go
val, err := /* something */
if sink.Error(err) {
	return
	// An error segment will be displayed automatically.
}
sink.Output(/* something with val */)
```

# Module

The building block for the bar, a `bar.Module` is anything that can send output to a `bar.Sink`. A
Module simply needs a `Stream(bar.Sink)` function that sends output to the passed-in sink any number
of times with arbitrary delays in between.

The passed in sink *may* become invalid once the Stream function returns, so avoid storing it
anywhere. Once Stream returns, it can be called again and the module should sensibly restart.

# Events

Event handlers (`func(bar.Event)`) can be attached to a segment. When i3bar sends an event, barista
internally uses the `Name` field to look up the click handler and sends the rest of the event to
the handler:

```go
type Event struct {
	Button  Button
	// Co-ordinates within the segment.
	X, Y int
	// Size of the segment as displayed on the bar.
	Width, Height int
	// Co-ordinates on the root window (useful for positioning popups).
	ScreenX, ScreenY int
}
```

# Error Handling

Right-clicks on a segment with a defined error are not sent to the segment's click handler, but
instead to barista's global error handler. By default, the global error handler uses `i3-nagbar` to
display the full error message, but it can be customised by calling `barista.SetErrorHandler`.

The global error handler receives the event as well as the error from the segment clickced:

```go
type ErrorEvent {
	Error error
	Event
}
```
