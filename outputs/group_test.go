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

package outputs

import (
	"testing"

	"github.com/soumya92/barista/bar"
	"github.com/stretchrcom/testify/assert"
)

func TestSegmentGroup(t *testing.T) {
	assert := assert.New(t)
	out := Group(
		bar.TextSegment("1"),
		bar.TextSegment("2"),
		bar.TextSegment("3"),
		bar.PangoSegment("4"),
		bar.PangoSegment("5"),
		bar.PangoSegment("6"),
	)

	// Since property changes don't propagate to the backing segments,
	// we'll fetch a new instance for each assertion.
	first := func() bar.Segment { return out.Segments()[0] }
	mid := func() bar.Segment { return out.Segments()[3] }
	last := func() bar.Segment { return out.Segments()[5] }

	assert.Equal("1", first().Text())
	assert.False(first().IsPango())

	assert.Equal("4", mid().Text())
	assert.True(mid().IsPango())

	assert.Equal("6", last().Text())
	assert.True(last().IsPango())

	assertAllEqual := func(expected interface{},
		getFunc func(s bar.Segment) (interface{}, bool),
		message string) {
		for _, segment := range out.Segments() {
			val, isSet := getFunc(segment)
			assert.True(isSet)
			assert.Equal(expected, val, message)
		}
	}

	out.Border(bar.Color("green"))
	assertAllEqual(bar.Color("green"),
		func(s bar.Segment) (interface{}, bool) { return s.GetBorder() },
		"sets border for all segments")

	out.Urgent(true)
	assertAllEqual(true,
		func(s bar.Segment) (interface{}, bool) { return s.IsUrgent() },
		"sets border for all segments")

	out.Urgent(false)
	assertAllEqual(false,
		func(s bar.Segment) (interface{}, bool) { return s.IsUrgent() },
		"sets border for all segments")

	sumMinWidth := func() int {
		minWidth := 0
		for _, s := range out.Segments() {
			m, _ := s.GetMinWidth()
			minWidth += m.(int)
		}
		return minWidth
	}

	out.MinWidth(60)
	assertAllEqual(10,
		func(s bar.Segment) (interface{}, bool) { return s.GetMinWidth() },
		"min_width when equally distributed")
	assert.Equal(60, sumMinWidth(), "min_width when equally distributed")

	// Test that however the min_width distribution happens, the sum of segments'
	// min_width should be whatever was given to the output as a whole.
	for _, testWidth := range []int{10, 100, 6, 3, 2, 1} {
		out.MinWidth(testWidth)
		assert.Equal(testWidth, sumMinWidth(), "min_width = %d", testWidth)
	}

	out.MinWidth(0)
	assertAllEqual(0,
		func(s bar.Segment) (interface{}, bool) { return s.GetMinWidth() },
		"min_width when 0")
	assert.Equal(0, sumMinWidth(), "min_width when 0")

	out.Separator(true)
	_, isSet := first().HasSeparator()
	assert.False(isSet, "separator only affects last segment")
	_, isSet = mid().HasSeparator()
	assert.False(isSet, "separator only affects last segment")
	sep, isSet := last().HasSeparator()
	assert.True(isSet, "separator only affects last segment")
	assert.True(sep)

	out.InnerSeparatorWidth(5)
	out.InnerSeparator(false)

	sep, isSet = first().HasSeparator()
	assert.True(isSet, "inner separator only affects inner segments")
	assert.False(sep)
	sepW, _ := first().GetSeparatorWidth()
	assert.Equal(5, sepW)
	sep, isSet = mid().HasSeparator()
	assert.True(isSet, "inner separator only affects inner segments")
	assert.False(sep)
	sepW, _ = mid().GetSeparatorWidth()
	assert.Equal(5, sepW)
	sep, _ = last().HasSeparator()
	assert.True(sep, "last segment separator untouched by inner separator")
	_, isSet = last().GetSeparatorWidth()
	assert.False(isSet, "last segment separator untouched by inner separator")
}

func TestSingleGroup(t *testing.T) {
	assert := assert.New(t)

	single := Group(bar.PangoSegment("<b>only</b>"))
	assert.Equal(1, len(single.Segments()))

	segment := func() bar.Segment { return single.Segments()[0] }
	assert.Equal("<b>only</b>", segment().Text())
	assert.True(segment().IsPango())

	single.Background(bar.Color("yellow"))
	bg, _ := segment().GetBackground()
	assert.Equal(bar.Color("yellow"), bg)

	single.Color(bar.Color("red"))
	col, _ := segment().GetColor()
	assert.Equal(bar.Color("red"), col)

	single.Align(bar.AlignEnd)
	align, _ := segment().GetAlignment()
	assert.Equal(bar.AlignEnd, align)

	single.SeparatorWidth(2)
	// Single segment should ignore inner separator.
	single.InnerSeparator(false)
	single.InnerSeparatorWidth(12)

	sepW, _ := segment().GetSeparatorWidth()
	assert.Equal(2, sepW)

	_, isSet := segment().HasSeparator()
	assert.False(isSet)

	single.MinWidth(100)
	minW, _ := segment().GetMinWidth()
	assert.Equal(100, minW)

	newLast := Text("this is now the last one").Color(bar.Color("cyan"))
	// If another segment is added, some properties must be adjusted.
	single.Append(newLast)

	// Min width is split between the two.
	minW, _ = segment().GetMinWidth()
	assert.Equal(50, minW)
	// Now using inner separator config.
	sepW, _ = segment().GetSeparatorWidth()
	assert.Equal(12, sepW)
	_, isSet = segment().HasSeparator()
	assert.True(isSet)

	segment = func() bar.Segment { return single.Segments()[1] }

	// The new segment should inherit any unset properties.
	bg, _ = segment().GetBackground()
	assert.Equal(bar.Color("yellow"), bg)

	// But retain previously set properties.
	col, _ = segment().GetColor()
	assert.Equal(bar.Color("cyan"), col)

	// And not set properties that weren't set on either.
	_, isSet = segment().GetBorder()
	assert.False(isSet)
}

func TestEmptyGroup(t *testing.T) {
	// Sanity check properties where the number of segments matters.
	empty := Group()
	empty.MinWidth(100)
	empty.Separator(true)
	empty.SeparatorWidth(0)
	empty.InnerSeparator(false)
	empty.InnerSeparatorWidth(10)
	// Make sure nothing blows up...
	assert.NotPanics(t, func() { empty.Segments() })
}

func TestMinWidthDistributions(t *testing.T) {
	assert := assert.New(t)
	out := Group(
		Text("1").MinWidth(100),
		Text("2").MinWidthPlaceholder("###.##"),
		Text("3").MinWidth(50),
		Text("4"),
	)

	minWidths := func() []interface{} {
		var m []interface{}
		for _, s := range out.Segments() {
			width, _ := s.GetMinWidth()
			m = append(m, width)
		}
		return m
	}

	// Min Width cannot fit the existing segments.
	out.MinWidth(100)
	assert.Equal([]interface{}{100, "###.##", 50, nil}, minWidths())

	// Min Width is exactly equal to existing segments.
	out.MinWidth(150)
	assert.Equal([]interface{}{100, "###.##", 50, 0}, minWidths())

	out.Append(Text("5"))
	assert.Equal([]interface{}{100, "###.##", 50, 0, 0}, minWidths())

	// Min Width is split between unset segments.
	out.MinWidth(230)
	assert.Equal([]interface{}{100, "###.##", 50, 40, 40}, minWidths())

	// Additional segments are added, min width should redistribute.
	out.Append(Text("6"), Text("7"))
	assert.Equal([]interface{}{100, "###.##", 50, 20, 20, 20, 20}, minWidths())
}
