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
	for k, v := range s.actual.i3map() {
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

	segment.Background(Color("green"))
	a.Expected["background"] = "green"
	a.AssertEqual("sets background color")

	segment.Border(Color("yellow"))
	a.Expected["border"] = "yellow"
	a.AssertEqual("sets border color")

	segment.Align(AlignStart)
	a.Expected["align"] = "left"
	a.AssertEqual("alignment strings are preserved")

	segment.MinWidth(10)
	a.Expected["min_width"] = "10"
	a.AssertEqual("sets min width in px")

	segment.MinWidthPlaceholder("00:00")
	a.Expected["min_width"] = "00:00"
	a.AssertEqual("sets min width placeholder")

	// sanity check default go values.
	segment.Separator(false)
	a.Expected["separator"] = "false"
	a.AssertEqual("separator = false")

	segment.Padding(0)
	a.Expected["separator_block_width"] = "0"
	a.AssertEqual("separator width = 0")

	segment.Urgent(false)
	a.Expected["urgent"] = "false"
	a.AssertEqual("urgent = false")

	segment.Identifier("ident")
	a.Expected["instance"] = "ident"
	a.AssertEqual("opaque instance")

	barOut := segment.Segments()
	assert.Equal(t, 1, len(barOut), "bar.Output from Segment returns 1 segment")
	assert.Equal(t, segment, barOut[0])
}

func TestGets(t *testing.T) {
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
