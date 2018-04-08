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

package bar

import (
	"fmt"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

type sA struct {
	*testing.T
	actual   Segment
	Expected map[string]string
}

func (s sA) AssertEqual(message string) {
	actualMap := make(map[string]string)
	for k, v := range s.actual {
		actualMap[k] = fmt.Sprintf("%v", v)
	}
	assert.Equal(s.T, s.Expected, actualMap, message)
}

func segmentAssertions(t *testing.T, segment Segment) sA {
	return sA{t, segment, make(map[string]string)}
}

func TestSegment(t *testing.T) {
	segment := TextSegment("test")
	a := segmentAssertions(t, segment)

	a.Expected["full_text"] = "test"
	a.Expected["markup"] = "none"
	a.AssertEqual("sets full_text")

	segment2 := segment.ShortText("t")
	a2 := segmentAssertions(t, segment2)
	a2.Expected["full_text"] = "test"
	a2.Expected["short_text"] = "t"
	a2.Expected["markup"] = "none"
	a2.AssertEqual("sets short_text, does not lose full_text")

	assert.Equal(t, "test", segment.Text(), "text getter")
	assert.Equal(t, "test", segment2.Text(), "text getter")

	a.Expected["short_text"] = "t"
	a.AssertEqual("mutates in place")

	segment.Color(Color("red"))
	a.Expected["color"] = "red"
	a.AssertEqual("sets color value")

	segment.Color(Color(""))
	delete(a.Expected, "color")
	a.AssertEqual("clears color value when blank")

	segment.Background(Color(""))
	a.AssertEqual("clearing unset color works")

	segment.Align(AlignStart)
	a.Expected["align"] = "left"
	a.AssertEqual("alignment strings are preserved")

	// sanity check default go values.
	segment.Separator(false)
	a.Expected["separator"] = "false"
	a.AssertEqual("separator = false")

	segment.SeparatorWidth(0)
	a.Expected["separator_block_width"] = "0"
	a.AssertEqual("separator width = 0")

	segment.Instance("instance")
	a.Expected["instance"] = "instance"
	a.AssertEqual("opaque instance")
}

func TestGroup(t *testing.T) {
	out := SegmentGroup{
		TextSegment("1"),
		TextSegment("2"),
		TextSegment("3"),
		PangoSegment("4"),
		PangoSegment("5"),
		PangoSegment("6"),
	}

	first := segmentAssertions(t, out[0])
	first.Expected["full_text"] = "1"
	first.Expected["markup"] = "none"

	mid := segmentAssertions(t, out[3])
	mid.Expected["full_text"] = "4"
	mid.Expected["markup"] = "pango"

	last := segmentAssertions(t, out[5])
	last.Expected["full_text"] = "6"
	last.Expected["markup"] = "pango"

	assertAllEqual := func(message string) {
		first.AssertEqual(message)
		mid.AssertEqual(message)
		last.AssertEqual(message)
	}

	assertAllEqual("initial values")

	out.Border(Color("green"))
	first.Expected["border"] = "green"
	mid.Expected["border"] = "green"
	last.Expected["border"] = "green"
	assertAllEqual("sets border for all segments")

	out.Urgent(true)
	first.Expected["urgent"] = "true"
	mid.Expected["urgent"] = "true"
	last.Expected["urgent"] = "true"
	assertAllEqual("sets urgent for all segments")

	out.Urgent(false)
	first.Expected["urgent"] = "false"
	mid.Expected["urgent"] = "false"
	last.Expected["urgent"] = "false"
	assertAllEqual("sets urgent for all segments")

	sumMinWidth := func() int {
		minWidth := 0
		for _, s := range out {
			minWidth += s["min_width"].(int)
		}
		return minWidth
	}

	out.MinWidth(60)
	first.Expected["min_width"] = "10"
	mid.Expected["min_width"] = "10"
	last.Expected["min_width"] = "10"
	assertAllEqual("min_width when equally distributed")
	assert.Equal(t, 60, sumMinWidth(), "min_width when equally distributed")

	// Test that however the min_width distribution happens, the sum of segments'
	// min_width should be whatever was given to the output as a whole.
	for _, testWidth := range []int{10, 100, 6, 3, 2, 1} {
		out.MinWidth(testWidth)
		assert.Equal(t, testWidth, sumMinWidth(), "min_width = %d", testWidth)
	}

	out.MinWidth(0)
	first.Expected["min_width"] = "0"
	mid.Expected["min_width"] = "0"
	last.Expected["min_width"] = "0"
	assertAllEqual("min_width when 0")
	assert.Equal(t, 0, sumMinWidth(), "min_width when 0")

	out.Separator(true)
	last.Expected["separator"] = "true"
	assertAllEqual("separator only affects last segment")

	out.InnerSeparatorWidth(5)
	out.InnerSeparator(false)
	first.Expected["separator_block_width"] = "5"
	first.Expected["separator"] = "false"
	mid.Expected["separator_block_width"] = "5"
	mid.Expected["separator"] = "false"
	assertAllEqual("inner separator only affects inner segments")

	single := SegmentGroup{PangoSegment("<b>only</b>")}
	a := segmentAssertions(t, single[0])
	a.Expected["full_text"] = "<b>only</b>"
	a.Expected["markup"] = "pango"
	single.Background(Color("yellow"))
	a.Expected["background"] = "yellow"
	single.Color(Color("red"))
	a.Expected["color"] = "red"
	single.Align(AlignEnd)
	a.Expected["align"] = "right"
	single.SeparatorWidth(2)
	a.Expected["separator_block_width"] = "2"
	single.MinWidth(100)
	a.Expected["min_width"] = "100"
	a.AssertEqual("setting properties on a single segment output work")

	// Sanity check properties where the number of segments matters.
	empty := SegmentGroup{}
	empty.MinWidth(100)
	empty.Separator(true)
	empty.SeparatorWidth(0)
	empty.InnerSeparator(false)
	empty.InnerSeparatorWidth(10)
}
