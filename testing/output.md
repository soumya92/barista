---
title: testing/Output
import_as: outputTest
---

The `output` testing package provides methods to make assertions against a bar.Output or a segment
within it. Construct a new `Assertions` object using `outputTest.New(t, out)`.

- `AssertEqual(output)`: Asserts that the segments match those in the given output.

- `AssertText([]string)`: Asserts that the text of the segments match the given strings.

- `AssertEmpty()`: Asserts that the output is empty (has no segments).

- `AssertError() []string`: Asserts that all segments are errors, and returns the error strings.

- `At(i)`: Returns `SegmentAssertions` for the segment at position `i` (0-based).

`SegmentAssertions` provides assertions against a single segment.

- `AssertEqual`/`AssertText`/`AssertError`: behave similarly to their output conterparts.

- `LeftClick()`: Simulates a left click on the segment by triggering the OnClick handler.

- `Click(bar.Event)`: Triggers the OnClick handler with the given Event.
