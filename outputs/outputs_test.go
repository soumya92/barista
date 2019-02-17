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

package outputs

import (
	"fmt"
	"io"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/format"
	"barista.run/pango"
	pangoTesting "barista.run/testing/pango"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"
)

func textOf(out bar.Output) string {
	str := ""
	for _, segment := range out.Segments() {
		txt, _ := segment.Content()
		str += txt
	}
	return str
}

func textWithSeparators(out bar.Output) string {
	str := ""
	for _, segment := range out.Segments() {
		txt, _ := segment.Content()
		str += txt
		if sep, _ := segment.HasSeparator(); sep {
			str += "|"
		}
	}
	return str
}

func TestTextFmt(t *testing.T) {
	tests := []struct {
		desc     string
		output   bar.Output
		expected string
	}{
		{"empty string", Text(""), ""},
		{"simple string", Text("test"), "test"},
		{"percent sign without interpolation", Text("100%"), "100%"},
		{"no interpolation", Textf("test"), "test"},
		{"with string args", Textf("%s=%s", "a", "b"), "a=b"},
		{"with multiple args", Textf("%s=%0.4f, %d^2=%d", "pi", 3.14159, 2, 4), "pi=3.1416, 2^2=4"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.expected, textOf(tc.output), tc.desc)
	}
}

func mustFormat(thing interface{}) format.Values {
	val, _ := format.Unit(thing)
	return val
}

func TestPango(t *testing.T) {
	// Most of pango is already tested by pango tests,
	// so we'll just test collapsing here.
	tests := []struct {
		desc     string
		output   bar.Output
		expected string
	}{
		{"empty", Pango(), ""},
		{"empty string", Pango(""), ""},
		{"simple string", Pango("test"), "test"},
		{"with attribute", Pango(pango.Text("test").Bold()), "<span weight='bold'>test</span>"},
		{
			"with pango node, string, and other",
			Pango(pango.Text("<").Heavy(), 3.14159, " ", true, pango.Text(">").Heavy()),
			"<span weight='heavy'>&lt;</span>3.14159 true<span weight='heavy'>&gt;</span>",
		},
		{
			"with units of various kinds",
			Pango(4*unit.Kilometer, " ", format.SI(3300, "g"), " ", 85*time.Minute, " ", mustFormat(4000*unit.Second)),
			`4.0<small>km</small> 3.3<small>kg</small> 1<small>h</small>25<small>m</small> 1<small>h</small> 6<small>m</small>`,
		},
	}
	for _, tc := range tests {
		pangoTesting.AssertEqual(t, tc.expected, textOf(tc.output), tc.desc)
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		desc     string
		output   bar.Output
		expected string
	}{
		{"well known error", Error(io.EOF), io.EOF.Error()},

		{
			"manually constructed error",
			Error(fmt.Errorf("error string")),
			"error string",
		},
		{
			"errorf with arguments",
			Errorf("cannot add %d and %s", 1, "a"),
			"cannot add 1 and a",
		},
	}
	for _, tc := range tests {
		err := tc.output.Segments()[0].GetError()
		require.Error(t, err, "Segment has associated error")
		require.Contains(t, err.Error(), tc.expected)
		require.Equal(t, textOf(tc.output), "Error", "Text is set to 'Error'")
		shortText, _ := tc.output.Segments()[0].GetShortText()
		require.Equal(t, shortText, "!", "Short text is set to '!'")
		urgent, _ := tc.output.Segments()[0].IsUrgent()
		require.True(t, urgent, "error is marked urgent")
	}
}

func TestGroup(t *testing.T) {
	tests := []struct {
		desc     string
		output   bar.Output
		expected string
	}{
		{"empty output", Group(), ""},

		{
			"simple group",
			Group(Text("1"), Textf("%d", 2)),
			"1|2|",
		},

		{
			"group with append",
			Group().Append(Text("1")).Append(Textf("%d", 2)),
			"1|2|",
		},

		{
			"without inner separators",
			Group(Text("1"), Textf("%d", 2)).InnerSeparators(false),
			"12|",
		},

		{
			"setting inner separators before adding modules",
			Group().InnerSeparators(false).Append(Text("1")).Append(Text("2")),
			"12|",
		},

		{
			"innerseparator with existing separators in modules",
			Group().
				InnerSeparators(false).
				Append(Text("1")).
				Append(Text("2").Separator(true)).
				Append(Textf("%d", 3)),
			"12|3|",
		},

		{
			"with explicitly removed separators",
			Group(
				Text("1"),
				Text("2").Separator(false),
				Textf("%d", 3)),
			"1|23|",
		},

		{
			"nested group with inner separators",
			Group(
				Text("1"),
				Group(
					Text("2").Separator(true),
					Text("3").Separator(false),
					Text("4"),
					Text("5").Separator(true),
				).InnerSeparators(false),
				Textf("%d", 6),
			).InnerSeparators(true),
			"1|2|345|6|",
		},
	}
	for _, tc := range tests {
		require.Equal(t, tc.expected, textWithSeparators(tc.output), tc.desc)
	}
}
