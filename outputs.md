---
title: Outputs
---

The `outputs` package provides some utility methods for simpler construction of common outputs.

- `Text(string)`: Constructs a bar.Output with a single text-only segment using the given string.

- `Textf(format string, ...interface{})`: Constructs a text segment with the interpolated result.

- `Error(error)`: Constructs a bar.Output with a single error segment.

- `Errorf(format string, ...interface{})`: Constructs an error segment with the interpolated result.

- `Pango(...interface{})`: Constructs a pango output from the given "things". Each value is
  processed as below:
  - `*pango.Node`: added directly
  - `string`: added as a `pango.Text()` node
  - everything else: added as `pango.Textf("%v", thing)`

## Group

`Group(...bar.Output)` creates a flattened group of segments from the individual segments of each
output, and allows appending further outputs to build up long outputs one portion at a time.

- `Append(bar.Output)`: adds all the segments of the given output to the end of the group.

- `Glue()`: removes all inner separators and separator widths, "gluing" the segments together
  seamlessly when displayed on the bar.

Most of the `*Segment` methods are also available on an output group, and they apply even to
segments added after the methods are called (e.g. `Color(red)` followed by `Append(foo)` will
make `foo` red as well).

These methods only set the property for segments that haven't already specified it, so something
like `Background(blue)` can be overridden for an individual segment using `Background(green)`.

For a complete list of methods, see the [SegmentGroup godoc](https://godoc.org/github.com/soumya92/barista/outputs#SegmentGroup).

## Datasize and Datarate formatting

> TODO: Move this somewhere else, outputs isn't really a good fit.

- `outputs.Bytesize(unit.Datasize)` formats the given datasize using [go-humanize](https://godoc.org/github.com/dustin/go-humanize).
- `outputs.IBytesize(unit.Datasize)` formats the datasize in IEC units.

- `outputs.Byterate(unit.Datarate)` formats the given datarate by appending "/s" to the formatted bytesize.
- `outputs.IByterate(unit.Datarate)` formats the given datarate in IEC units.
