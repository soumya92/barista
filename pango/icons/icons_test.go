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
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/pango"
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
		assert.Nil(t, err, "No error decoding a valid symbol from hex")
		assert.Equal(t, tc.expected, value, "Symbol is decoded correctly")
	}

	_, err := SymbolFromHex("04xc")
	assert.Error(t, err, "Error with invalid hex string")
	_, err = SymbolFromHex("ffffffffffff")
	assert.Error(t, err, "Error with out of bounds hex")
}

func TestIconProvider(t *testing.T) {
	testIcons := provider{
		symbols: map[string]string{
			"test":          "a",
			"lgtm":          "👍",
			"ligature-font": "home",
		},
		attrs: []pango.Attribute{
			pango.Font("testfont"),
			pango.Weight(200),
		},
	}

	tests := []struct{ desc, icon, expected string }{
		{"no output for unknown icon", "unknown", ""},
		{"simple icon", "test", "<span face='testfont' weight='200'>a</span>"},
		{"emoji", "lgtm", "<span face='testfont' weight='200'>👍</span>"},
		{"ligature", "ligature-font", "<span face='testfont' weight='200'>home</span>"},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.expected, testIcons.Icon(tc.icon).Pango(), tc.desc)
	}

	assert.Equal(t,
		"<span color='#ff0000' face='testfont' weight='200'>a</span>",
		testIcons.Icon("test", colors.Hex("#f00")).Pango(),
		"Additional attributes when requesting an icon",
	)

	assert.Equal(t,
		"", testIcons.Icon("notfound", colors.Hex("#f00")).Pango(),
		"Empty even with additional attributes when requesting a non-existent icon",
	)

	assert.Equal(t,
		"<span weight='bold' face='testfont'>a</span>",
		testIcons.Icon("test", pango.Bold).Pango(),
		"Override default attributes when named the same",
	)
}
