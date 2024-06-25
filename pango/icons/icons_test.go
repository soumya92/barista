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

	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/pango"
	pangoTesting "github.com/soumya92/barista/testing/pango"

	"github.com/stretchr/testify/require"
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
		require.NoError(t, err, "Decoding a valid symbol from hex")
		require.Equal(t, tc.expected, value, "Symbol is decoded correctly")
	}

	_, err := SymbolFromHex("04xc")
	require.Error(t, err, "Error with invalid hex string")
	_, err = SymbolFromHex("ffffffffffff")
	require.Error(t, err, "Error with out of bounds hex")
}

func TestIconProvider(t *testing.T) {
	p := NewProvider("test")
	require.NoError(t, p.Hex("lgtm", "1F44D"))
	require.Error(t, p.Hex("not-real", "xx"))
	p.Symbol("test", "a")
	p.Symbol("ligature-font", "home")
	p.Font("testfont")
	p.AddStyle(func(n *pango.Node) { n.Weight(200) })

	tests := []struct{ desc, icon, expected string }{
		{"no output for unknown icon", "unknown", ""},
		{"simple icon", "test", "<span fallback='false' face='testfont' weight='200'>a</span>"},
		{"emoji", "lgtm", "<span fallback='false' face='testfont' weight='200'>👍</span>"},
		{"ligature", "ligature-font", "<span fallback='false' face='testfont' weight='200'>home</span>"},
	}
	for _, tc := range tests {
		pangoTesting.AssertEqual(t, tc.expected, pango.Icon("test-"+tc.icon).String(), tc.desc)
	}

	pangoTesting.AssertEqual(t,
		"<span color='#ff0000'><span fallback='false' face='testfont' weight='200'>a</span></span>",
		pango.Icon("test-test").Color(colors.Hex("#f00")).String(),
		"Attributes are added to a wrapping <span>",
	)

	pangoTesting.AssertEqual(t,
		`<span style='italic'><span fallback='false' weight='200' face='testfont'>home</span
		><span weight='bold'>foobar</span></span>`,
		pango.Icon("test-ligature-font").Italic().Append(pango.Text("foobar").Bold()).String(),
		"Append adds new elements without icon font styling",
	)
}
