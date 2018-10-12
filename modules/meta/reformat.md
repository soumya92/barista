---
title: Reformat
---

Reformat provides a module that modifies the output from an existing module before sending it to the
bar. It also provides utility functions to make transformations easier.

Reformatting an existing module: `reformat.New(existingModule).Format(formatFunc)`.

- `type FormatFunc = func(bar.Segments) bar.Output` operates at the output level.
- `type SegmentFunc = func(*bar.Segment) *bar.Segment` operates at the segment level.

## Default Formatters

There are some formatting functions available out of the box:
- `Original`: a.k.a. Identity. Returns the input unchanged.
- `Hide`: Returns nil, hiding the original module from the bar.
- `Texts(func(string) string)`: Transforms only text segments, replacing them with new text segments.
- `EachSegment(SegmentFunc)`: Transforms each segment of the original output individually.

In addition, reformat also provides `SkipErrors(SegmentFunc)` that wraps an existing SegmentFunc
with code to return error segments unchanged, potentially simplifying the transformation code.

## Example

<div class="module-example-out"><span>Error</span><span>**Liftoff!**</span></div>
Adding '**' around all non-error output:

```go
// countdown is a module that outputs an error and a countdown.
reformat.New(countdown).Format(
	reformat.EachSegment(
		reformat.SkipErrors(func(in *bar.Segment) *bar.Segment {
			txt, _ := in.Content()
			return in.Text("**" + txt + "**")
		})))
```
