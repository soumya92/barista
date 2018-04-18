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
	"testing"

	"github.com/stretchrcom/testify/assert"
)

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

	segment.Color(Color("red"))
	assert.Equal(Color("red"), assertSet(segment.GetColor()))

	segment.Background(Color("green"))
	assert.Equal(Color("green"), assertSet(segment.GetBackground()))

	segment.Border(Color("yellow"))
	assert.Equal(Color("yellow"), assertSet(segment.GetBorder()))

	segment.Urgent(true)
	assert.True(assertSet(segment.IsUrgent()).(bool))

	segment.Separator(false)
	assert.False(assertSet(segment.HasSeparator()).(bool))

	segment.Padding(3)
	assert.Equal(3, assertSet(segment.GetPadding()))

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
}

func TestBarOutput(t *testing.T) {
	segment := TextSegment("test").Align(AlignCenter)
	barOut := segment.Segments()
	assert.Equal(t, 1, len(barOut), "bar.Output from Segment returns 1 segment")
	assert.Equal(t, segment, barOut[0])
}

func TestClone(t *testing.T) {
	assert := assert.New(t)
	a := TextSegment("10 deg C").
		Urgent(true).
		MinWidthPlaceholder("## deg C")
	b := a.Clone()

	assert.Equal(a, b, "copied values are the same")
	c := b.Background(Color("green"))

	assert.NotEqual(a, b, "changes to b not reflected in a")
	_, isSet := a.GetBackground()
	assert.False(isSet)
	bg, isSet := b.GetBackground()
	assert.True(isSet)
	assert.Equal(Color("green"), bg)

	c.ShortText("short")
	assert.Equal(b, c, "chained methods still return same segment")
	text, isSet := b.GetShortText()
	assert.True(isSet)
	assert.Equal("short", text)
}
