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
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/pango"
)

func TestEmpty(t *testing.T) {
	assert.Empty(t, Empty(), "empty output")
}

func textOf(out bar.Output) string {
	str := ""
	for _, segment := range out {
		str += segment["full_text"].(string)
	}
	return str
}

func textWithSeparators(out bar.Output) string {
	str := ""
	for _, segment := range out {
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
		assert.Equal(t, bar.MarkupPango, tc.output[0]["markup"].(bar.Markup), tc.desc)
	}
}

var testObject = struct {
	Number   int
	Text     string
	Fraction float64
	HTML     string
	Object   struct{ YesNo bool }
}{
	Number:   42,
	Text:     "test-string",
	Fraction: 2.7182818,
	HTML:     "<b>bold</b>",
	Object:   struct{ YesNo bool }{YesNo: true},
}

func TestTextTemplate(t *testing.T) {
	assert.Panics(t, func() { TextTemplate("{{invalid template") }, "panic on invalid template")
	assert.NotPanics(t, func() { TextTemplate("string") }, "no panic on simple string")
	assert.NotPanics(t, func() { TextTemplate("number = {{.number}}") }, "no panic on simple template")

	tests := []struct {
		desc     string
		template TemplateFunc
		expected string
	}{
		{"simple template", TextTemplate(`{{.Number}}`), "42"},
		{"multiple fields", TextTemplate(`{{.Number}} {{.Text}}`), "42 test-string"},
		{"piping through formatter", TextTemplate(`{{.Fraction | printf "%.4f"}}`), "2.7183"},
		{"if/else", TextTemplate(`{{if .Object.YesNo}}yes{{else}}no{{end}}`), "yes"},
		{
			"pango markup not interpreted",
			TextTemplate(`<span size='{{.Number}}'>{{.HTML}}</span>`),
			"<span size='42'><b>bold</b></span>",
		},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, textOf(tc.template(testObject)), tc.desc)
	}
}

func TestPangoTemplate(t *testing.T) {
	assert.Panics(t, func() { PangoTemplate("{{invalid template") }, "panic on invalid template")
	// TODO: Handle invalid markup.
	//assert.Panics(t, func() { PangoTemplate("<unclosed tag") }, "panic on unclosed tag")
	//assert.Panics(t, func() { PangoTemplate("<tag invalid='attribute>") }, "panic on unclosed attribute")
	//assert.Panics(t, func() { PangoTemplate("<a></b>") }, "panic on mismatched tags")
	assert.NotPanics(t, func() { PangoTemplate("string") }, "no panic on simple string")
	assert.NotPanics(t, func() { PangoTemplate("number = {{.number}}") }, "no panic on simple template")

	tests := []struct {
		desc     string
		template TemplateFunc
		expected string
	}{
		{"simple template", PangoTemplate(`{{.Number}}`), "42"},
		{"multiple fields", PangoTemplate(`{{.Number}} <b>{{.Text}}</b>`), "42 <b>test-string</b>"},
		{"piping through formatter", PangoTemplate(`{{.Fraction | printf "%.4f"}}`), "2.7183"},
		{"if/else", PangoTemplate(`{{if .Object.YesNo}}yes{{else}}no{{end}}`), "yes"},
		{
			"pango markup escaped",
			PangoTemplate(`<span size='{{.Number}}'>{{.HTML}}</span>`),
			"<span size='42'>&lt;b&gt;bold&lt;/b&gt;</span>",
		},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, textOf(tc.template(testObject)), tc.desc)
	}
}

func TestComposite(t *testing.T) {
	tests := []struct {
		desc     string
		output   bar.Output
		expected string
	}{
		{"empty output", Multi().Build(), ""},

		{
			"simple composite",
			Multi().AddText("name1", "1").AddTextf("name2", "%d", 2).Build(),
			"12|",
		},

		{
			"composite with separator",
			Multi().AddText("n1", "1").AddText("n2", "2").KeepSeparators(true).Build(),
			"1|2|",
		},

		{
			"calling KeepSeparators before adding modules",
			Multi().KeepSeparators(true).AddText("n1", "1").AddText("n2", "2").Build(),
			"1|2|",
		},

		{
			"KeepSeparators with explicitly removed separators",
			Multi().
				KeepSeparators(true).
				AddText("name1", "1").
				Add("name2", bar.Output{bar.NewSegment().Separator(false).Text("2")}).
				AddTextf("name3", "%d", 3).
				Build(),
			"1|23|",
		},

		{
			"with explicitly added separators",
			Multi().
				AddText("name1", "1").
				Add("name2", bar.Output{bar.NewSegment().Separator(true).Text("2")}).
				AddTextf("name3", "%d", 3).
				Build(),
			"12|3|",
		},

		{
			"Built from multi-segment bar.Output",
			Multi().
				AddText("name1", "1").
				Add("name2", bar.Output{
					bar.NewSegment().Separator(true).Text("2"),
					bar.NewSegment().Separator(false).Text("3"),
					bar.NewSegment().Text("4"),
					bar.NewSegment().Separator(true).Text("5"),
				}).
				AddTextf("name3", "%d", 6).
				Build(),
			"12|345|6|",
		},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, textWithSeparators(tc.output), tc.desc)
	}
}
