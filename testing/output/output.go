// Copyright 2018 Google Inc.
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

// Package output provides utilities for testing barista outputs and segments.
package output

import (
	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
)

// New creates an object that provides assertions on a bar.Output,
// such as equality, emptiness, text equality, and error state across
// all its segments.
func New(t require.TestingT, out bar.Output) Assertions {
	return Assertions{
		output:  out,
		require: require.New(t),
	}
}

// Assertions provides assertions that simplify testing outputs.
type Assertions struct {
	output  bar.Output
	require *require.Assertions
}

// AssertEqual asserts that the actual output contains exactly the
// same segments as the expected output.
func (a Assertions) AssertEqual(expected bar.Output, args ...interface{}) {
	if !a.Expect(args...) {
		return
	}
	a.require.Equal(expected.Segments(), a.output.Segments(), args...)
}

// AssertEmpty asserts that the actual output has no segments.
func (a Assertions) AssertEmpty(args ...interface{}) {
	if !a.Expect(args...) {
		return
	}
	a.require.Empty(a.output.Segments(), args...)
}

// AssertText asserts that the text of each segment matches the
// expected value.
func (a Assertions) AssertText(expected []string, args ...interface{}) {
	if !a.Expect(args...) {
		return
	}
	segments := a.output.Segments()
	actual := make([]string, len(segments))
	for i, s := range segments {
		actual[i] = s.Text()
	}
	a.require.Equal(expected, actual, args...)
}

// AssertError asserts that each segment in the output is an error,
// and returns a slice containing the error descriptions.
func (a Assertions) AssertError(args ...interface{}) []string {
	if !a.Expect(args...) {
		return nil
	}
	segments := a.output.Segments()
	if len(segments) == 0 {
		a.require.Fail("Expected error, got no output", args...)
		return nil
	}
	texts := make([]string, len(segments))
	for i := range segments {
		texts[i] = a.At(i).AssertError(args...)
	}
	return texts
}

// Expect asserts that the output is not nil. Used in a chain, e.g.
// testBar.LatestOutput().Expect("expected an output")
func (a Assertions) Expect(args ...interface{}) bool {
	if a.output == nil {
		a.require.Fail("Expected an output, got nil", args...)
		return false
	}
	return true
}

// At creates segment assertions for the segment at position i.
// It fails the test if there are not enough segments.
func (a Assertions) At(i int) SegmentAssertions {
	if !a.Expect() {
		return SegmentAssertions{require: a.require}
	}
	segments := a.output.Segments()
	if i >= len(segments) {
		a.require.Fail("Not enough segments",
			"want #%d, have %d", i, len(segments))
		return SegmentAssertions{require: a.require}
	}
	return SegmentAssertions{segment: segments[i], require: a.require}
}

// Len returns the number of segments in the actual output.
func (a Assertions) Len() int {
	if !a.Expect() {
		return 0
	}
	return len(a.output.Segments())
}

// Segment provides text, error, and equality assertions for a bar.Segment
func Segment(t require.TestingT, segment *bar.Segment) SegmentAssertions {
	if segment == nil {
		require.Fail(t, "Asserting against nil segment")
	}
	return SegmentAssertions{
		segment: segment,
		require: require.New(t),
	}
}

// SegmentAssertions provides assertions that simplify testing individual
// segments within an output.
type SegmentAssertions struct {
	segment *bar.Segment
	require *require.Assertions
}

// AssertEqual asserts that the actual segment is equal to the expecte segment.
func (a SegmentAssertions) AssertEqual(expected *bar.Segment, args ...interface{}) {
	if a.segment == nil {
		return
	}
	a.require.Equal(expected, a.segment, args...)
}

// AssertText asserts that the segment's text matches the expected string.
func (a SegmentAssertions) AssertText(expected string, args ...interface{}) {
	if a.segment == nil {
		return
	}
	a.require.Equal(expected, a.segment.Text(), args...)
}

// AssertError asserts that the segment represents an error,
// and returns the error description.
func (a SegmentAssertions) AssertError(args ...interface{}) string {
	if a.segment == nil {
		return ""
	}
	err := a.segment.GetError()
	if err == nil {
		a.require.Fail("expected an error", args...)
		return ""
	}
	return err.Error()
}

// Segment returns the actual segment to allow fine-grained assertions.
// This is doubly useful because Assertions.At(i) returns SegmentAssertions,
// allowing code like:
//     urgent, _ := out.At(2).Segment().IsUrgent()
//     require.True(t, urgent, "segment #3 is urgent")
func (a SegmentAssertions) Segment() *bar.Segment {
	return a.segment
}
