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

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/pango"
)

func TestEmpty(t *testing.T) {
	assert.Empty(t, Empty().Segments(), "empty output")
}

func textOf(out bar.Output) string {
	str := ""
	for _, segment := range out.Segments() {
		str += segment["full_text"].(string)
	}
	return str
}

func textWithSeparators(out bar.Output) string {
	str := ""
	for _, segment := range out.Segments() {
		str += segment["full_text"].(string)
		if sep, ok := segment["separator"]; !ok || sep.(bool) {
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
		assert.Equal(t, tc.expected, textOf(tc.output), tc.desc)
	}
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
		{"with attribute", Pango(pango.Bold, "test"), "<span weight='bold'>test</span>"},
		{"with tag", Pango(pango.Tag("b", "test")), "<b>test</b>"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, textOf(tc.output), tc.desc)
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

		{
			"error in pango template",
			PangoTemplate(`<b>{{.NoSuchField}}</b>`)(testObject),
			"can't evaluate field NoSuchField",
		},

		{
			"error in text template",
			TextTemplate(`{{.Number.Nested}}`)(testObject),
			"can't evaluate field Nested in type int",
		},
	}
	for _, tc := range tests {
		assert.Contains(t, textOf(tc.output), tc.expected, tc.desc)
		assert.Equal(t, tc.output.Segments()[0]["short_text"], "Error",
			"Short text is set to 'Error'")
		assert.True(t, tc.output.Segments()[0]["urgent"].(bool),
			"error is marked urgent")
	}
}

func TestGroup(t *testing.T) {
	tests := []struct {
		desc     string
		output   bar.Output
		expected string
	}{
		{"empty", Group(), ""},

		{
			"simple group",
			Group(Text("1"), Textf("%d", 2)),
			"1|2|",
		},

		{
			"group without inner separator",
			Group(Text("1"), Textf("%d", 2)).InnerSeparator(false),
			"12|",
		},

		{
			"nested grouping",
			Group(
				Text("1").Separator(false),
				Group(
					bar.TextSegment("2").Separator(true),
					bar.TextSegment("3").Separator(false),
					bar.TextSegment("4").Separator(false),
					bar.TextSegment("5").Separator(true),
				),
				Textf("%d", 6),
			),
			"12|345|6|",
		},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, textWithSeparators(tc.output), tc.desc)
	}
}
