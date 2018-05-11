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
	"image/color"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"
)

func assertColorEquals(t *testing.T, expected, actual color.Color, args ...interface{}) {
	if expected == nil {
		assert.Nil(t, actual, args...)
		return
	}
	var e, a struct{ r, g, b, a uint32 }
	e.r, e.g, e.b, e.a = expected.RGBA()
	a.r, a.g, a.b, a.a = actual.RGBA()
	assert.Equal(t, e, a, args...)
}

func TestCreation(t *testing.T) {
	scheme["test"] = Hex("#abcdef")
	scheme["empty"] = nil

	creationTests := []struct {
		desc     string
		color    color.Color
		expected color.Color
	}{
		{"simple hex color", Hex("#001122"), color.RGBA{0x00, 0x11, 0x22, 0xff}},
		{"short hex color", Hex("#07f"), color.RGBA{0x00, 0x77, 0xff, 0xff}},
		{"invalid hex color", Hex("#ghi"), nil},
		{"scheme empty", Scheme("empty"), nil},
		{"scheme color", Scheme("test"), color.RGBA{0xab, 0xcd, 0xef, 0xff}},
		{"scheme non-existent", Scheme("undefined"), nil},
	}

	for _, tc := range creationTests {
		assertColorEquals(t, tc.expected, tc.color, tc.desc)
	}
}

func assertSchemeEquals(t *testing.T, expected map[string]color.Color, desc string) {
	for name, expectedValue := range expected {
		assertColorEquals(t, expectedValue, Scheme(name), desc)
	}
	for name, value := range scheme {
		assertColorEquals(t, expected[name], value, desc)
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
		scheme = map[string]color.Color{}
		LoadFromArgs(tc.args)
		assert.Empty(t, scheme, tc.desc)
	}

	schemeTests := []struct {
		desc     string
		args     []string
		expected map[string]color.Color
	}{
		{
			"simple arg",
			[]string{"color1=#ff0000"},
			map[string]color.Color{"color1": Hex("#ff0000")},
		},
		{
			"multiple args",
			[]string{"color1=#abcdef", "color2=#00ff00"},
			map[string]color.Color{"color1": Hex("#abcdef"), "color2": Hex("#00ff00")},
		},
		{
			"mixed args",
			[]string{"color1=#abc", "color2=#00ff00", "color3=invalid"},
			map[string]color.Color{"color1": Hex("#aabbcc"), "color2": Hex("#00ff00")},
		},
		{
			"non-color args",
			[]string{"color1=#abc", "--debugmode", "--logtofile=/var/log/bar"},
			map[string]color.Color{"color1": Hex("#aabbcc")},
		},
	}

	for _, tc := range schemeTests {
		scheme = map[string]color.Color{}
		LoadFromArgs(tc.args)
		assertSchemeEquals(t, tc.expected, tc.desc)
	}
}

func TestLoadFromMap(t *testing.T) {
	schemeTests := []struct {
		desc     string
		args     map[string]string
		expected map[string]color.Color
	}{
		{"empty args", map[string]string{}, map[string]color.Color{}},
		{
			"simple arg",
			map[string]string{"color1": "#ff0000"},
			map[string]color.Color{"color1": Hex("#ff0000")},
		},
		{
			"multiple args",
			map[string]string{"color1": "#abcdef", "color2": "#00ff00"},
			map[string]color.Color{
				"color1": color.RGBA{0xab, 0xcd, 0xef, 0xff},
				"color2": color.RGBA{0x00, 0xff, 0x00, 0xff},
			},
		},
		{
			"invalid args",
			map[string]string{"color": "invalid", "other": "#ghi", "invalid": "vkdl32"},
			map[string]color.Color{},
		},
		{
			"mixed args",
			map[string]string{"color1": "#abc", "color2": "#00ff00", "color3": "invalid"},
			map[string]color.Color{
				"color1": color.RGBA{0xaa, 0xbb, 0xcc, 0xff},
				"color2": color.RGBA{0x00, 0xff, 0x00, 0xff},
			},
		},
	}

	for _, tc := range schemeTests {
		scheme = map[string]color.Color{}
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
		expected map[string]color.Color
	}{
		{"empty", map[string]color.Color{}},
		{"no-colors", map[string]color.Color{}},
		{"simple", map[string]color.Color{"good": Hex("#007700")}},
		{"mixed", map[string]color.Color{
			"good": Hex("#007700"),
			"bad":  Hex("#ff0000"),
			"1":    Hex("#0000ff"),
			"2":    Hex("#abcdef"),
		}},
	}

	for _, tc := range schemeTests {
		scheme = map[string]color.Color{}
		err := LoadFromConfig(tc.file)
		assert.Nil(t, err)
		assertSchemeEquals(t, tc.expected, tc.file)
	}
}
