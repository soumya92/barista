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

## Timed Outputs

`outputs.Repeat` allows construction of repeating outputs using the `bar.TimedOutput` extension. A
repeating output is constructed using a `func(time.Time) bar.Output`, with a set repeating strategy:

- `Every(time.Duration)`: Calls the function at a fixed interval.

- `AtNext(time.Duration)`: Calls the function whenever the current time is a multiple of the given
  duration. For example, `AtNext(time.Minute)` will call the function at `14:00:00`, `14:01:00`, and
  so on.

- `At(...time.Time)`: Calls the function at each of the times specified.

`outputs.AtDeltaFrom` allows construction of a `bar.TimedOutput` that counts down to, or up from,
a reference point in time. It is constructed from a `func(time.Duration) bar.Output`, with the
argument being positive for reference points in the past, and negative for points in the future.

- `From(time.Time)`/`FromFine(time.Time)`: Calls the function based on the time remaining
  until or elapsed since the given `time.Time`:

  |---------------------|--------------|--------------|
  | Remaining / Elapsed | `From`       | `FromFine`   |
  |---------------------|--------------|--------------|
  | < 1 minute          | Every second | Every second |
  | < 1 hour            | Every minute | Every second |
  | < 24 hours          | Every hour   | Every minute |
  | > 24 hours          | Every hour   | Every hour   |
  |---------------------|--------------|--------------|


### Implementation notes:

- The function will be called with a time value that exactly matches the rule used, even if the real
  time has skewed a bit.

- The function may be called multiple times for the same `time.Time`, especially when used in a
  group. The function must be idempotent.

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
