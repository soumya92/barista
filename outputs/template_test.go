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

	"github.com/martinlindhe/unit"
	"github.com/soumya92/barista/bar"
	"github.com/stretchr/testify/require"
)

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
	require.Panics(t, func() { TextTemplate("{{invalid template") }, "panic on invalid template")
	require.NotPanics(t, func() { TextTemplate("string") }, "no panic on simple string")
	require.NotPanics(t, func() { TextTemplate("number = {{.number}}") }, "no panic on simple template")

	tests := []struct {
		desc     string
		template func(interface{}) bar.Output
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
		require.Equal(t, tc.expected, textOf(tc.template(testObject)), tc.desc)
	}
}

var unitsObject = struct {
	Size unit.Datasize
	Rate unit.Datarate
}{
	Size: 10 * unit.Kilobyte,
	Rate: 192 * unit.KilobitPerSecond,
}

func TestTemplateFuncs(t *testing.T) {
	tests := []struct {
		desc     string
		template string
		expected string
	}{
		{"bytesize", `{{.Size | bytesize}}`, "10 kB"},
		{"ibytesize", `{{.Size | ibytesize}}`, "9.8 KiB"},
		{"byterate", `{{.Rate | byterate}}`, "24 kB/s"},
		{"ibyterate", `{{.Rate | ibyterate}}`, "23 KiB/s"},
	}
	for _, tc := range tests {
		require.Equal(t, tc.expected, textOf(TextTemplate(tc.template)(unitsObject)), tc.desc)
	}
}
