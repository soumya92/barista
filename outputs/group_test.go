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
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/stretchr/testify/require"
)

func TestSegmentGroup(t *testing.T) {
	require := require.New(t)
	evtCh := make(chan bar.Event)
	out := Group(
		bar.TextSegment("1").OnClick(nil),
		bar.TextSegment("2"),
		bar.TextSegment("3"),
		bar.PangoSegment("4").OnClick(func(e bar.Event) { evtCh <- e }),
		bar.PangoSegment("5"),
		bar.PangoSegment("6"),
	)

	// Since property changes don't propagate to the backing segments,
	// we'll fetch a new instance for each assertion.
	first := func() *bar.Segment { return out.Segments()[0] }
	mid := func() *bar.Segment { return out.Segments()[3] }
	last := func() *bar.Segment { return out.Segments()[5] }

	txt, isPango := first().Content()
	require.Equal("1", txt)
	require.False(isPango)

	txt, isPango = mid().Content()
	require.Equal("4", txt)
	require.True(isPango)

	txt, isPango = last().Content()
	require.Equal("6", txt)
	require.True(isPango)

	assertAllEqual := func(expected interface{},
		getFunc func(s *bar.Segment) (interface{}, bool),
		message string) {
		for _, segment := range out.Segments() {
			val, isSet := getFunc(segment)
			require.True(isSet)
			require.Equal(expected, val, message)
		}
	}

	out.Border(colors.Hex("#070"))
	assertAllEqual(colors.Hex("#070"),
		func(s *bar.Segment) (interface{}, bool) { return s.GetBorder() },
		"sets border for all segments")

	out.Urgent(true)
	assertAllEqual(true,
		func(s *bar.Segment) (interface{}, bool) { return s.IsUrgent() },
		"sets border for all segments")

	out.Urgent(false)
	assertAllEqual(false,
		func(s *bar.Segment) (interface{}, bool) { return s.IsUrgent() },
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
		func(s *bar.Segment) (interface{}, bool) { return s.GetMinWidth() },
		"min_width when equally distributed")
	require.Equal(60, sumMinWidth(), "min_width when equally distributed")

	// Test that however the min_width distribution happens, the sum of segments'
	// min_width should be whatever was given to the output as a whole.
	for _, testWidth := range []int{10, 100, 6, 3, 2, 1} {
		out.MinWidth(testWidth)
		require.Equal(testWidth, sumMinWidth(), "min_width = %d", testWidth)
	}

	out.MinWidth(0)
	assertAllEqual(0,
		func(s *bar.Segment) (interface{}, bool) { return s.GetMinWidth() },
		"min_width when 0")
	require.Equal(0, sumMinWidth(), "min_width when 0")

	out.Separator(true)
	_, isSet := first().HasSeparator()
	require.False(isSet, "separator only affects last segment")
	_, isSet = mid().HasSeparator()
	require.False(isSet, "separator only affects last segment")
	sep, isSet := last().HasSeparator()
	require.True(isSet, "separator only affects last segment")
	require.True(sep)

	out.InnerPadding(5)
	out.InnerSeparators(false)

	sep, isSet = first().HasSeparator()
	require.True(isSet, "inner separator only affects inner segments")
	require.False(sep)
	pad, _ := first().GetPadding()
	require.Equal(5, pad)
	sep, isSet = mid().HasSeparator()
	require.True(isSet, "inner separator only affects inner segments")
	require.False(sep)
	pad, _ = mid().GetPadding()
	require.Equal(5, pad)
	sep, _ = last().HasSeparator()
	require.True(sep, "last segment separator untouched by inner separator")
	_, isSet = last().GetPadding()
	require.False(isSet, "last segment separator untouched by inner separator")

	groupEvtCh := make(chan bar.Event)
	out.OnClick(func(e bar.Event) { groupEvtCh <- e })

	go first().Click(bar.Event{X: 10})
	select {
	case <-groupEvtCh:
		require.Fail("Click event triggered with nil handler")
	case <-time.After(10 * time.Millisecond):
	}

	go mid().Click(bar.Event{Y: 20})
	select {
	case e := <-evtCh:
		require.Equal(bar.Event{Y: 20}, e)
	case <-time.After(time.Second):
		require.Fail("Click event not received in preset handler")
	case <-groupEvtCh:
		require.Fail("Click event triggered with preset handler")
	}

	go last().Click(bar.Event{X: 40})
	select {
	case e := <-groupEvtCh:
		require.Equal(bar.Event{X: 40}, e)
	case <-time.After(time.Second):
		require.Fail("Click event not received in preset handler")
	}

	out.Glue()
	pad, _ = mid().GetPadding()
	require.Equal(0, pad, "Glue removes inner padding")
	sep, _ = mid().HasSeparator()
	require.Equal(false, sep, "Glue removes inner separator")
}

func TestSingleGroup(t *testing.T) {
	require := require.New(t)

	single := Group(bar.PangoSegment("<b>only</b>"))
	require.Equal(1, len(single.Segments()))

	segment := func() *bar.Segment { return single.Segments()[0] }

	txt, isPango := segment().Content()
	require.Equal("<b>only</b>", txt)
	require.True(isPango)

	single.Background(colors.Hex("#ff0"))
	bg, _ := segment().GetBackground()
	require.Equal(colors.Hex("#ff0"), bg)

	single.Color(colors.Hex("#f00"))
	col, _ := segment().GetColor()
	require.Equal(colors.Hex("#f00"), col)

	single.Align(bar.AlignEnd)
	align, _ := segment().GetAlignment()
	require.Equal(bar.AlignEnd, align)

	single.Padding(2)
	// Single segment should ignore inner separator.
	single.InnerSeparators(false)
	single.InnerPadding(12)

	pad, _ := segment().GetPadding()
	require.Equal(2, pad)

	_, isSet := segment().HasSeparator()
	require.False(isSet)

	single.MinWidth(100)
	minW, _ := segment().GetMinWidth()
	require.Equal(100, minW)

	newLast := Text("this is now the last one").Color(colors.Hex("#0ff"))
	// If another segment is added, some properties must be adjusted.
	single.Append(newLast)

	// Min width is split between the two.
	minW, _ = segment().GetMinWidth()
	require.Equal(50, minW)
	// Now using inner separator config.
	pad, _ = segment().GetPadding()
	require.Equal(12, pad)
	_, isSet = segment().HasSeparator()
	require.True(isSet)

	segment = func() *bar.Segment { return single.Segments()[1] }

	// The new segment should inherit any unset properties.
	bg, _ = segment().GetBackground()
	require.Equal(colors.Hex("#ff0"), bg)

	// But retain previously set properties.
	col, _ = segment().GetColor()
	require.Equal(colors.Hex("#0ff"), col)

	// And not set properties that weren't set on either.
	_, isSet = segment().GetBorder()
	require.False(isSet)
}

func TestEmptyGroup(t *testing.T) {
	// Sanity check properties where the number of segments matters.
	empty := Group()
	empty.MinWidth(100)
	empty.Separator(true)
	empty.Padding(0)
	empty.InnerSeparators(false)
	empty.InnerPadding(10)
	// Make sure nothing blows up...
	require.NotPanics(t, func() { empty.Segments() })
}

func TestMinWidthDistributions(t *testing.T) {
	require := require.New(t)
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
	require.Equal([]interface{}{100, "###.##", 50, nil}, minWidths())

	// Min Width is exactly equal to existing segments.
	out.MinWidth(150)
	require.Equal([]interface{}{100, "###.##", 50, 0}, minWidths())

	out.Append(Text("5"))
	require.Equal([]interface{}{100, "###.##", 50, 0, 0}, minWidths())

	// Min Width is split between unset segments.
	out.MinWidth(230)
	require.Equal([]interface{}{100, "###.##", 50, 40, 40}, minWidths())

	// Additional segments are added, min width should redistribute.
	out.Append(Text("6")).Append(Text("7"))
	require.Equal([]interface{}{100, "###.##", 50, 20, 20, 20, 20}, minWidths())
}
