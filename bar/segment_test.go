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

	"github.com/stretchrcom/testify/assert"
)

func assertColorEqual(t *testing.T, expected, actual color.Color, args ...interface{}) {
	var e, a struct{ r, g, b, a uint32 }
	e.r, e.g, e.b, e.a = expected.RGBA()
	a.r, a.g, a.b, a.a = actual.RGBA()
	assert.Equal(t, e, a, args...)
}

func TestSegment(t *testing.T) {
	assert := assert.New(t)

	segment := TextSegment("test")
	assert.Equal("test", segment.Text())
	assert.False(segment.IsPango())

	assertUnset := func(value interface{}, isSet bool) interface{} {
		assert.False(isSet)
		return value
	}

	assertSet := func(value interface{}, isSet bool) interface{} {
		assert.True(isSet)
		return value
	}

	assertUnset(segment.GetShortText())
	assertUnset(segment.GetAlignment())
	assertUnset(segment.GetColor())
	assertUnset(segment.GetBackground())
	assertUnset(segment.GetBorder())
	assertUnset(segment.GetMinWidth())
	assertUnset(segment.GetID())

	defaultUrgent := assertUnset(segment.IsUrgent())
	assert.False(defaultUrgent.(bool))

	defaultSep := assertUnset(segment.HasSeparator())
	assert.True(defaultSep.(bool))

	defaultSepWidth := assertUnset(segment.GetPadding())
	assert.Equal(9, defaultSepWidth)

	segment = PangoSegment("<b>bold</b>")
	assert.Equal("<b>bold</b>", segment.Text())
	assert.True(segment.IsPango())

	assertUnset(segment.GetShortText())
	segment.ShortText("BD")
	assert.Equal("BD", assertSet(segment.GetShortText()))
	segment.ShortText("")
	assert.Equal("", assertSet(segment.GetShortText()))

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
	assert.True(assertSet(segment.IsUrgent()).(bool))

	segment.Separator(false)
	assert.False(assertSet(segment.HasSeparator()).(bool))

	segment.Padding(3)
	assert.Equal(3, assertSet(segment.GetPadding()))

	segment.Error(errors.New("foo"))
	assert.Error(segment.GetError())

	segment.Error(nil)
	assert.NoError(segment.GetError())

	segment.MinWidth(40)
	assert.Equal(40, assertSet(segment.GetMinWidth()))
	segment.MinWidth(0)
	assert.Equal(0, assertSet(segment.GetMinWidth()))

	segment.MinWidthPlaceholder("00:00:00")
	assert.Equal("00:00:00", assertSet(segment.GetMinWidth()))
	segment.MinWidthPlaceholder("")
	assert.Equal("", assertSet(segment.GetMinWidth()))

	segment.Identifier("test")
	assert.Equal("test", assertSet(segment.GetID()))

	segment = ErrorSegment(fmt.Errorf("something went wrong"))
	assert.Equal("Error", segment.Text())
	assert.Equal("!", assertSet(segment.GetShortText()))
	assert.True(assertSet(segment.IsUrgent()).(bool))
	assert.Error(segment.GetError())
	assertUnset(segment.GetMinWidth())
	segment.MinWidthPlaceholder("error")
	assert.Equal("error", assertSet(segment.GetMinWidth()))
}

func TestBarOutput(t *testing.T) {
	segment := TextSegment("test").Align(AlignCenter)
	barOut := segment.Segments()
	assert.Equal(t, 1, len(barOut), "bar.Output from Segment returns 1 segment")
	assert.Equal(t, segment, barOut[0])

	segment0 := TextSegment("foo")
	segment1 := TextSegment("baz")
	segments := Segments{segment0, segment1}
	barOut = segments.Segments()
	assert.Equal(t, 2, len(barOut), "bar.Output from Segments returns all segments")
	assert.Equal(t, segment0, barOut[0])
	assert.Equal(t, segment1, barOut[1])
}

func TestClone(t *testing.T) {
	assert := assert.New(t)
	a := TextSegment("10 deg C").
		Urgent(true).
		MinWidthPlaceholder("## deg C")
	b := a.Clone()

	assert.Equal(a, b, "copied values are the same")
	c := b.Background(color.White)

	assert.NotEqual(a, b, "changes to b not reflected in a")
	_, isSet := a.GetBackground()
	assert.False(isSet)
	bg, isSet := b.GetBackground()
	assert.True(isSet)
	assertColorEqual(t, color.Gray{0xff}, bg)

	c.ShortText("short")
	assert.Equal(b, c, "chained methods still return same segment")
	text, isSet := b.GetShortText()
	assert.True(isSet)
	assert.Equal("short", text)
}
