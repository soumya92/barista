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

package pango

import (
	"image/color"
	"testing"
	"time"

	"barista.run/base/value"
	"barista.run/colors"
	"barista.run/format"
	"barista.run/testing/output"
	"barista.run/testing/pango"

	"github.com/stretchr/testify/require"
)

var stringifyingTests = []struct {
	desc     string
	node     *Node
	expected string
}{
	{"zero value", &Node{}, ""},
	{"empty element", New(), ""},
	{"append text", Text("foo").AppendText("bar"), "foobar"},
	{
		"empty element with attribute",
		New().Weight(400),
		"<span weight='400'></span>",
	},
	{
		"zero value can add attributes and content",
		(&Node{}).Oblique().AppendText("foo", "bar"),
		"<span style='oblique'>foobar</span>",
	},
	{
		"text with attribute",
		Text("text").UltraBold(),
		"<span weight='ultrabold'>text</span>",
	},
	{
		"append text and set attribute",
		Text("foo").AppendText("bar").UltraLight().Italic(),
		"<span weight='ultralight' style='italic'>foobar</span>",
	},
	{
		"append styled text",
		Text("foo").Append(Text("bar").XXLarge().StyleNormal()),
		"foo<span size='xx-large' style='normal'>bar</span>",
	},
	{
		"repeated relative size",
		Text("tiny").Smaller().Smaller().Smaller().Append(Text(" tot").SmallCaps()),
		"<small><small><small>tiny<span variant='smallcaps'> tot</span></small></small></small>",
	},
	{
		"append styled text with attributes",
		Text("foo").Font("monospace").Append(Text("bar").Heavy().Strikethrough()).AppendText("baz"),
		"<span face='monospace'>foo<span strikethrough='true' weight='heavy'>bar</span>baz</span>",
	},

	{
		"multiple append",
		Text("").
			Light().
			AppendText("con", "cat").
			Append(Textf(": %.3f", 3.141)).
			Append(Text("u").UnderlineSingle()).LetterSpacing(1),
		"<span weight='light' letter_spacing='1024'>concat: 3.141<span underline='single'>u</u></b>",
	},

	{
		"text with special characters",
		Text("<>&amp;'\"=").Expanded(),
		"<span stretch='expanded'>&lt;&gt;&amp;amp;&#39;&#34;=</span>",
	},
	{
		"text with 'valid' html",
		Text("<b color='red'>bold</b>").Oblique(),
		"<span style='oblique'>&lt;b color=&#39;red&#39;&gt;bold&lt;/b&gt;</span>",
	},

	{
		"unnamed tag collapsing",
		New(Text("string-1"), Text(" "), Text("string-2")).AppendTextf(" %.3f", 2.718),
		"string-1 string-2 2.718",
	},
	{
		"mixing fixed and relative sizes",
		Text("foo").Size(10.0).Larger().Larger().Font("monospace").AppendText("bar"),
		"<span size='10240' face='monospace'><big><big>foobar</big></big></span>",
	},
	{
		"package example #0",
		New(
			Text("Red "),
			Text("Bold Text").Bold()).
			Color(colors.Hex("#ff0000")),
		`<span color="#ff0000">Red <span weight="bold">Bold Text</span></span>`,
	},
	{
		"package example #1",
		Text("Red ").
			Color(colors.Hex("#ff0000")).
			Append(Text("Bold Text").Bold()),
		`<span color="#ff0000">Red <span weight="bold">Bold Text</span></span>`,
	},
	{
		"parent example #0",
		Text("c").Condensed().Color(colors.Hex("#ff0000")).
			Concat(Text("foo")).UnderlineError(),
		"<span underline='error'><span stretch='condensed' color='#ff0000'>c</span>foo</span>",
	},
	{
		"concat helper methods",
		Text("c").Condensed().Color(colors.Hex("#ff0000")).
			ConcatText("foo", "bar").UnderlineError().
			ConcatTextf(" - %.2f", 3.14159),
		"<span underline='error'><span stretch='condensed' color='#ff0000'>c</span>foobar</span> - 3.14",
	},
	{
		"complex",
		complex(),
		`<span weight='600' rise='400' size='14336' face='monospace'
		><span face='serif' weight='ultrabold'>Number 42</span
		><span underline='double' size='small'>small underline</span
		><span stretch='ultraexpanded' size='x-large' style='oblique' variant='smallcaps'
		>all the styles!</span></span>`,
	},
	{
		"different values for same style",
		Text("normal").Bold().UltraLight().Weight(40).WeightNormal().
			SmallCaps().VariantNormal().
			UnderlineError().UnderlineLow().UnderlineNone().
			Italic().Oblique().StyleNormal().
			Strikethrough().NoStrikethrough().
			UltraCondensed().ExtraExpanded().ExtraCondensed().SemiCondensed().SemiExpanded().StretchNormal().
			XSmall().XXSmall().Medium().Large().Size(10.0),
		`<span
			weight='normal'
			variant='normal'
			underline='none'
			style='normal'
			strikethrough='false'
			stretch='normal'
			size='10240'>normal</span>`,
	},
	{
		"unit",
		Unit(format.SI(44000, "m")),
		` 44<small>km</small>`,
	},
	{
		"unit with decimal",
		Unit(format.SI(4400, "m")),
		`4.4<small>km</small>`,
	},
	{
		"multiple units",
		Unit(format.Duration(4*time.Hour + 5*time.Minute)...),
		`4<small>h</small> 5<small>m</small>`,
	},
}

func TestStringifying(t *testing.T) {
	for _, tc := range stringifyingTests {
		pango.AssertEqual(t, tc.expected, tc.node.String(), tc.desc)
	}
}

func TestCustomUnitFormatter(t *testing.T) {
	defer func() { unitFormatter = value.Value{} }()
	SetUnitFormatter(func(v format.Values) *Node {
		return Textf(v.String())
	})

	pango.AssertEqual(t, `4.40km`,
		Unit(format.SI(4400, "m")).String())
	pango.AssertEqual(t, `1d20h3m`,
		Unit(format.SI(1, "d"), format.SI(20, "h"), format.SI(3.2, "m")).String())
}

var transparent = color.Transparent
var solid = color.White
var partial = color.RGBA64{0x7fff, 0x0, 0x0, 0x7fff}

var colorAttrTests = []struct {
	desc     string
	node     *Node
	expected string
}{
	{
		"fg, transparent",
		Text("color").Color(transparent),
		"<span alpha='0'>color</span>",
	},
	{
		"bg, transparent",
		Text("color").Background(transparent),
		"<span background_alpha='0'>color</span>",
	},
	{
		"underline, transparent",
		Text("color").UnderlineColor(transparent),
		"<span>color</span>",
	},
	{
		"strikethrough, transparent",
		Text("color").StrikethroughColor(transparent),
		"<span>color</span>",
	},

	{
		"fg, with alpha",
		Text("color").Color(partial),
		"<span alpha='32767' color='#ff0000'>color</span>",
	},
	{
		"bg, with alpha",
		Text("color").Background(partial),
		"<span background_alpha='32767' background='#ff0000'>color</span>",
	},
	{
		"underline, with alpha",
		Text("color").UnderlineColor(partial),
		"<span underline_color='#ff0000'>color</span>",
	},
	{
		"strikethrough, with alpha",
		Text("color").StrikethroughColor(partial),
		"<span strikethrough_color='#ff0000'>color</span>",
	},

	{
		"fg, solid",
		Text("color").Color(solid),
		"<span color='#ffffff'>color</span>",
	},
	{
		"bg, solid",
		Text("color").Background(solid),
		"<span background='#ffffff'>color</span>",
	},
	{
		"underline, solid",
		Text("color").UnderlineColor(solid),
		"<span underline_color='#ffffff'>color</span>",
	},
	{
		"strikethrough, solid",
		Text("color").StrikethroughColor(solid),
		"<span strikethrough_color='#ffffff'>color</span>",
	},
	{
		"fg, nil",
		Text("color").Color(nil),
		"<span>color</span>",
	},
	{
		"bg, nil",
		Text("color").Background(nil),
		"<span>color</span>",
	},
	{
		"underline, nil",
		Text("color").UnderlineColor(nil),
		"<span>color</span>",
	},

	{
		"fg, alpha only, no colour",
		Text("dim").Alpha(0.5),
		"<span alpha='32768'>dim</span>",
	},
}

func TestColorAttrs(t *testing.T) {
	for _, tc := range colorAttrTests {
		pango.AssertEqual(t, tc.expected, tc.node.String(), tc.desc)
	}
}

func TestBarOutput(t *testing.T) {
	node := Text("something went wrong").Color(colors.Hex("#f00")).UnderlineError()
	segment := output.New(t, node).At(0).Segment()
	txt, isPango := segment.Content()
	pango.AssertEqual(t,
		"<span color='#ff0000' underline='error'>something went wrong</span>",
		txt)
	require.True(t, isPango)
}

var result string
var resultNode *Node

func benchmarkConstructOnly(b *testing.B, fn func() *Node) {
	var r *Node
	for n := 0; n < b.N; n++ {
		r = fn()
	}
	resultNode = r
}

func benchmarkConstructAndStringify(b *testing.B, fn func() *Node) {
	var s string
	for n := 0; n < b.N; n++ {
		s = fn().String()
	}
	result = s
}

func benchmarkStringifyOnly(b *testing.B, fn func() *Node) {
	var s string
	node := fn()
	for n := 0; n < b.N; n++ {
		s = node.String()
	}
	result = s
}

func empty() *Node                         { return New() }
func BenchmarkEmpty(b *testing.B)          { benchmarkConstructAndStringify(b, empty) }
func BenchmarkEmptyConstruct(b *testing.B) { benchmarkConstructOnly(b, empty) }
func BenchmarkEmptyStringify(b *testing.B) { benchmarkStringifyOnly(b, empty) }

func simple() *Node                         { return Text("text").Heavy() }
func BenchmarkSimple(b *testing.B)          { benchmarkConstructAndStringify(b, simple) }
func BenchmarkSimpleConstruct(b *testing.B) { benchmarkConstructOnly(b, simple) }
func BenchmarkSimpleStringify(b *testing.B) { benchmarkStringifyOnly(b, simple) }

func textonly() *Node                     { return Textf("%s-%d", "test", 1024) }
func BenchmarkText(b *testing.B)          { benchmarkConstructAndStringify(b, textonly) }
func BenchmarkTextConstruct(b *testing.B) { benchmarkConstructOnly(b, textonly) }
func BenchmarkTextStringify(b *testing.B) { benchmarkStringifyOnly(b, textonly) }

func complex() *Node {
	return New(
		Textf("Number %d", 42).UltraBold().Font("serif"),
		Textf("small underline").UnderlineDouble().Small(),
		Textf("%s %s %s!", "all", "the", "styles").UltraExpanded().Oblique().XLarge().SmallCaps(),
	).Font("monospace").Weight(600).Rise(400).Size(14.0)
}
func BenchmarkComplex(b *testing.B)          { benchmarkConstructAndStringify(b, complex) }
func BenchmarkComplexConstruct(b *testing.B) { benchmarkConstructOnly(b, complex) }
func BenchmarkComplexStringify(b *testing.B) { benchmarkStringifyOnly(b, complex) }
