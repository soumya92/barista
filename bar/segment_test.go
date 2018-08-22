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
	"errors"
	"fmt"
	"image/color"
	"testing"

	"github.com/stretchr/testify/require"
)

func assertColorEqual(t *testing.T, expected, actual color.Color, args ...interface{}) {
	var e, a struct{ r, g, b, a uint32 }
	e.r, e.g, e.b, e.a = expected.RGBA()
	a.r, a.g, a.b, a.a = actual.RGBA()
	require.Equal(t, e, a, args...)
}

func TestSegment(t *testing.T) {
	require := require.New(t)

	segment := TextSegment("test")
	txt, pango := segment.Content()
	require.Equal("test", txt)
	require.False(pango)

	segment.Pango("foo")
	txt, pango = segment.Content()
	require.Equal("foo", txt)
	require.True(pango)

	assertUnset := func(value interface{}, isSet bool) interface{} {
		require.False(isSet)
		return value
	}

	assertSet := func(value interface{}, isSet bool) interface{} {
		require.True(isSet)
		return value
	}

	assertUnset(segment.GetShortText())
	assertUnset(segment.GetAlignment())
	assertUnset(segment.GetColor())
	assertUnset(segment.GetBackground())
	assertUnset(segment.GetBorder())
	assertUnset(segment.GetMinWidth())
	require.False(segment.HasClick())

	defaultUrgent := assertUnset(segment.IsUrgent())
	require.False(defaultUrgent.(bool))

	defaultSep := assertUnset(segment.HasSeparator())
	require.True(defaultSep.(bool))

	defaultSepWidth := assertUnset(segment.GetPadding())
	require.Equal(9, defaultSepWidth)

	segment = PangoSegment("<b>bold</b>")
	txt, pango = segment.Content()
	require.Equal("<b>bold</b>", txt)
	require.True(pango)

	segment.Text("not-bold")
	txt, pango = segment.Content()
	require.Equal("not-bold", txt)
	require.False(pango)

	assertUnset(segment.GetShortText())
	segment.ShortText("BD")
	require.Equal("BD", assertSet(segment.GetShortText()))
	segment.ShortText("")
	require.Equal("", assertSet(segment.GetShortText()))

	segment.Color(color.Gray{0x77})
	assertColorEqual(t, color.RGBA{0x77, 0x77, 0x77, 0xff},
		assertSet(segment.GetColor()).(color.Color))

	segment.Background(color.RGBA{0x00, 0xff, 0x00, 0xff})
	assertColorEqual(t, color.RGBA{0x00, 0xff, 0x00, 0xff},
		assertSet(segment.GetBackground()).(color.Color))

	segment.Border(color.Transparent)
	assertColorEqual(t, color.RGBA{0, 0, 0, 0},
		assertSet(segment.GetBorder()).(color.Color))

	segment.Urgent(true)
	require.True(assertSet(segment.IsUrgent()).(bool))

	segment.Separator(false)
	require.False(assertSet(segment.HasSeparator()).(bool))

	segment.Padding(3)
	require.Equal(3, assertSet(segment.GetPadding()))

	segment.Error(errors.New("foo"))
	require.Error(segment.GetError())

	segment.Error(nil)
	require.NoError(segment.GetError())

	segment.MinWidth(40)
	require.Equal(40, assertSet(segment.GetMinWidth()))
	segment.MinWidth(0)
	require.Equal(0, assertSet(segment.GetMinWidth()))

	segment.MinWidthPlaceholder("00:00:00")
	require.Equal("00:00:00", assertSet(segment.GetMinWidth()))
	segment.MinWidthPlaceholder("")
	require.Equal("", assertSet(segment.GetMinWidth()))

	require.NotPanics(func() { segment.Click(Event{}) })
	segment.OnClick(nil)
	require.True(segment.HasClick())
	require.NotPanics(func() { segment.Click(Event{}) })
	var clickedEvent *Event
	segment.OnClick(func(e Event) { clickedEvent = &e })
	segment.Click(Event{Button: ButtonLeft})
	require.NotNil(clickedEvent)
	require.Equal(Event{Button: ButtonLeft}, *clickedEvent)

	segment = ErrorSegment(fmt.Errorf("something went wrong"))
	txt, pango = segment.Content()
	require.Equal("Error", txt)
	require.False(pango)
	require.Equal("!", assertSet(segment.GetShortText()))
	require.True(assertSet(segment.IsUrgent()).(bool))
	require.Error(segment.GetError())
	assertUnset(segment.GetMinWidth())
	segment.MinWidthPlaceholder("error")
	require.Equal("error", assertSet(segment.GetMinWidth()))
}

func TestBarOutput(t *testing.T) {
	segment := TextSegment("test").Align(AlignCenter)
	barOut := segment.Segments()
	require.Equal(t, 1, len(barOut), "bar.Output from Segment returns 1 segment")
	require.Equal(t, segment, barOut[0])

	segment0 := TextSegment("foo")
	segment1 := TextSegment("baz")
	segments := Segments{segment0, segment1}
	barOut = segments.Segments()
	require.Equal(t, 2, len(barOut), "bar.Output from Segments returns all segments")
	require.Equal(t, segment0, barOut[0])
	require.Equal(t, segment1, barOut[1])
}

func TestClone(t *testing.T) {
	require := require.New(t)
	a := TextSegment("10 deg C").
		Urgent(true).
		MinWidthPlaceholder("## deg C")
	b := a.Clone()

	require.Equal(a, b, "copied values are the same")
	c := b.Background(color.White)

	require.NotEqual(a, b, "changes to b not reflected in a")
	_, isSet := a.GetBackground()
	require.False(isSet)
	bg, isSet := b.GetBackground()
	require.True(isSet)
	assertColorEqual(t, color.Gray{0xff}, bg)

	c.ShortText("short")
	require.Equal(b, c, "chained methods still return same segment")
	text, isSet := b.GetShortText()
	require.True(isSet)
	require.Equal("short", text)
}
