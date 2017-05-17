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

package colors

import (
	"testing"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
)

func TestCreation(t *testing.T) {
	scheme["test"] = Hex("#abcdef")
	scheme["empty"] = Empty()

	creationTests := []struct {
		desc     string
		color    bar.Color
		expected string
	}{
		{"empty color", Empty(), ""},
		{"simple hex color", Hex("#001122"), "#001122"},
		{"short hex color", Hex("#07f"), "#0077ff"},
		{"invalid hex color", Hex("#ghi"), ""},
		{"colorful color from RGB", Colorful(colorful.Color{R: 1, G: 0.5, B: 0}), "#ff8000"},
		{"scheme empty", Scheme("empty"), ""},
		{"scheme color", Scheme("test"), "#abcdef"},
		{"scheme non-existent", Scheme("undefined"), ""},
	}

	for _, tc := range creationTests {
		assert.Equal(t, tc.expected, string(tc.color), tc.desc)
	}
}

func assertSchemeEquals(t *testing.T, expected map[string]string, desc string) {
	for name, expectedValue := range expected {
		assert.Equal(t, expectedValue, string(Scheme(name)), desc)
	}
	for name, value := range scheme {
		assert.Equal(t, expected[name], string(value), desc)
	}
}

func TestLoadFromArgs(t *testing.T) {
	emptySchemeTests := []struct {
		desc string
		args []string
	}{
		{"empty args", []string{}},
		{"no = in args", []string{"", "non-color args only", "blah test"}},
		{"invalid colors in args", []string{"color=invalid", "other=#ghi", "invalid=vkdl32"}},
	}

	for _, tc := range emptySchemeTests {
		scheme = map[string]bar.Color{}
		LoadFromArgs(tc.args)
		assert.Empty(t, scheme, tc.desc)
	}

	schemeTests := []struct {
		desc     string
		args     []string
		expected map[string]string
	}{
		{
			"simple arg",
			[]string{"color1=#ff0000"},
			map[string]string{"color1": "#ff0000"},
		},
		{
			"multiple args",
			[]string{"color1=#abcdef", "color2=#00ff00"},
			map[string]string{"color1": "#abcdef", "color2": "#00ff00"},
		},
		{
			"mixed args",
			[]string{"color1=#abc", "color2=#00ff00", "color3=invalid"},
			map[string]string{"color1": "#aabbcc", "color2": "#00ff00"},
		},
	}

	for _, tc := range schemeTests {
		scheme = map[string]bar.Color{}
		LoadFromArgs(tc.args)
		assertSchemeEquals(t, tc.expected, tc.desc)
	}
}

func TestLoadFromMap(t *testing.T) {
	schemeTests := []struct {
		desc     string
		args     map[string]string
		expected map[string]string
	}{
		{"empty args", map[string]string{}, map[string]string{}},
		{
			"simple arg",
			map[string]string{"color1": "#ff0000"},
			map[string]string{"color1": "#ff0000"},
		},
		{
			"multiple args",
			map[string]string{"color1": "#abcdef", "color2": "#00ff00"},
			map[string]string{"color1": "#abcdef", "color2": "#00ff00"},
		},
		{
			"invalid args",
			map[string]string{"color": "invalid", "other": "#ghi", "invalid": "vkdl32"},
			map[string]string{},
		},
		{
			"mixed args",
			map[string]string{"color1": "#abc", "color2": "#00ff00", "color3": "invalid"},
			map[string]string{"color1": "#aabbcc", "color2": "#00ff00"},
		},
	}

	for _, tc := range schemeTests {
		scheme = map[string]bar.Color{}
		LoadFromMap(tc.args)
		assertSchemeEquals(t, tc.expected, tc.desc)
	}
}

func TestLoadFromConfig(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "empty", []byte{}, 0644)
	afero.WriteFile(fs, "no-colors", []byte(`
general {
	output_format = "i3bar"
	colors = true
	interval = 5
}

order += "localtime"

localtime {
	format = "%H:%M"
}
`), 0644)
	afero.WriteFile(fs, "simple", []byte(`
general {
	output_format = "i3bar"
	colors = true
	color_good = "#007700"
}
`), 0644)
	afero.WriteFile(fs, "mixed", []byte(`
general {
	output_format = "i3bar"
	colors = true
	interval = 5
	color_bad = '#ff0000'
	color_good = "#007700"
	color_invalid = '#fhgkde'
	color_named = 'yellow'
	color_1='#00f'
	color_2  =    #abcdef
	colorignored = '#100'
	color_no_value
}
`), 0644)

	assert.Error(t, LoadFromConfig("non-existent"), "non-existent file")

	schemeTests := []struct {
		file     string
		expected map[string]string
	}{
		{"empty", map[string]string{}},
		{"no-colors", map[string]string{}},
		{"simple", map[string]string{"good": "#007700"}},
		{"mixed", map[string]string{
			"good": "#007700",
			"bad":  "#ff0000",
			"1":    "#0000ff",
			"2":    "#abcdef",
		}},
	}

	for _, tc := range schemeTests {
		scheme = map[string]bar.Color{}
		err := LoadFromConfig(tc.file)
		assert.Nil(t, err)
		assertSchemeEquals(t, tc.expected, tc.file)
	}
}
