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

package icons

import (
	"fmt"
	"io"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/pango"
	pangoTesting "github.com/soumya92/barista/testing/pango"
)

func TestSymbolFromHex(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"1F44D", "👍"},
		{"0024", "$"},
		{"20AC", "€"},
		{"10437", "𐐷"},
		{"24B62", "𤭢"},
	}
	for _, tc := range tests {
		value, err := SymbolFromHex(tc.input)
		assert.NoError(t, err, "Decoding a valid symbol from hex")
		assert.Equal(t, tc.expected, value, "Symbol is decoded correctly")
	}

	_, err := SymbolFromHex("04xc")
	assert.Error(t, err, "Error with invalid hex string")
	_, err = SymbolFromHex("ffffffffffff")
	assert.Error(t, err, "Error with out of bounds hex")
}

func TestIconProvider(t *testing.T) {
	p := NewProvider("test", Config{})
	p.symbols = map[string]string{
		"test":          "a",
		"lgtm":          "👍",
		"ligature-font": "home",
	}
	p.styler = func(n *pango.Node) {
		n.Font("testfont").Weight(200)
	}

	tests := []struct{ desc, icon, expected string }{
		{"no output for unknown icon", "unknown", ""},
		{"simple icon", "test", "<span face='testfont' weight='200'>a</span>"},
		{"emoji", "lgtm", "<span face='testfont' weight='200'>👍</span>"},
		{"ligature", "ligature-font", "<span face='testfont' weight='200'>home</span>"},
	}
	for _, tc := range tests {
		pangoTesting.AssertEqual(t, tc.expected, pango.Icon("test-"+tc.icon).Pango(), tc.desc)
	}

	pangoTesting.AssertEqual(t,
		"<span color='#ff0000'><span face='testfont' weight='200'>a</span></span>",
		pango.Icon("test-test").Color(colors.Hex("#f00")).Pango(),
		"Attributes are added to a wrapping <span>",
	)

	pangoTesting.AssertEqual(t,
		`<span style='italic'><span weight='200' face='testfont'>home</span
		><span weight='bold'>foobar</span></span>`,
		pango.Icon("test-ligature-font").Italic().Append(pango.Text("foobar").Bold()).Pango(),
		"Append adds new elements without icon font styling",
	)
}

func TestLoadingProviders(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "empty", []byte{}, 0644)
	afero.WriteFile(fs, "twoline", []byte(`
icon1
icon2

`), 0644)

	err := NewProvider("test", Config{Font: "testfont", FilePath: "non-existent"}).
		LoadFromFile(func(r io.Reader, addFunc func(string, string)) error {
			assert.Fail(t, "parseFunc is not called when file can't be opened")
			return nil
		})

	assert.Error(t, err, "error from reading file is propagated")
	assert.Empty(t,
		pango.Icon("test-icon1").Pango(),
		"empty pango markup returned for icon when file can't be opened",
	)

	err = NewProvider("test", Config{Font: "testfont", FilePath: "empty"}).
		LoadFromFile(func(r io.Reader, addFunc func(string, string)) error {
			addFunc("icon1", "random1")
			addFunc("icon2", "random2")
			return nil
		})

	assert.Nil(t, err, "no error when file is read and parse doesn't return one")
	pangoTesting.AssertEqual(t,
		"<span face='testfont'>random1</span>",
		pango.Icon("test-icon1").Pango(),
		"icon added in parseFile is correctly returned",
	)

	err = NewProvider("test", Config{Font: "testfont", FilePath: "empty"}).
		LoadFromFile(func(r io.Reader, addFunc func(string, string)) error {
			return fmt.Errorf("some error")
		})
	assert.Error(t, err, "error from parse is propagated")

	var lines []string
	err = NewProvider("test", Config{Font: "testfont", FilePath: "twoline"}).
		LoadByLines(func(line string, addFunc func(string, string)) error {
			lines = append(lines, line)
			addFunc(line, "filler")
			return nil
		})

	assert.Nil(t, err, "no error when file is read and parse doesn't return one")
	pangoTesting.AssertEqual(t,
		"<span face='testfont'>filler</span>",
		pango.Icon("test-icon2").Pango(),
		"icon added in parseLine is correctly returned",
	)
	assert.Contains(t, lines, "icon1", "all lines are parsed")
	assert.Contains(t, lines, "icon2", "all lines are parsed")
	assert.Equal(t, 2, len(lines), "Blank lines are ignored")

	err = NewProvider("test", Config{Font: "testfont", FilePath: "twoline"}).
		LoadByLines(func(line string, addFunc func(string, string)) error {
			return fmt.Errorf("some error")
		})
	assert.Error(t, err, "error from parse is propagated")

	NewProvider("test", Config{
		Font:     "testfont",
		FilePath: "twoline",
		Styler:   func(n *pango.Node) { n.Bold().Small() },
	}).LoadByLines(func(line string, addFunc func(string, string)) error {
		addFunc(line, "filler")
		return nil
	})
	pangoTesting.AssertEqual(t,
		"<span weight='bold' size='small' face='testfont'>filler</span>",
		pango.Icon("test-icon1").Pango(),
		"additional attributes in Config are added to provider's output",
	)
}
