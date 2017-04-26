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
	"fmt"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

type intAttribute int

func (m intAttribute) AttrName() string {
	return "int"
}

func (m intAttribute) AttrValue() string {
	return fmt.Sprintf("%d", int(m))
}

type customAttribute struct {
	name, value string
}

func (c customAttribute) AttrName() string {
	return c.name
}

func (c customAttribute) AttrValue() string {
	return c.value
}

var stringifyingTests = []struct {
	desc     string
	node     Node
	expected string
}{
	{"empty span", Span(), ""},
	{"empty not-span tag", Tag("b"), "<b></b>"},
	{"empty span with attribute", Tag("span", Weight(400)), "<span weight='400'></span>"},

	{"empty span()", Span(), ""},
	{"span() with attribute and text", Span(Bold, "text"), "<span weight='bold'>text</span>"},

	{"nested tags", Tag("i", Tag("b")), "<i><b></b></i>"},
	{
		"nested repeated tag",
		Tag("small", Tag("small", Tag("small"))),
		"<small><small><small></small></small></small>",
	},
	{
		"nested tags with attributes",
		Tag("span", Font("monospace"), Span(Bold)),
		"<span face='monospace'><span weight='bold'></span></span>",
	},

	{"int attribute", Tag("tt", intAttribute(5)), "<tt int='5'></tt>"},
	{"custom attribute", Tag("b", customAttribute{"name", "value"}), "<b name='value'></b>"},

	{"tag with text", Tag("b", "bold"), "<b>bold</b>"},
	{"tag with non-string child", Tag("b", 4.5), "<b>4.5</b>"},
	{
		"tag with text and attributes",
		Tag("span", Rise(400), "some text", Font("monospace")),
		"<span rise='400' face='monospace'>some text</span>",
	},
	{
		"tag with fmt-formatted text",
		Span(SemiExpanded, Textf("%03d", 4)),
		"<span stretch='semiexpanded'>004</span>",
	},
	{
		"tag with multiple children",
		Tag("b", "con", "cat", Font("monospace"), ": ", 3.141, Tag("u", "underline")),
		"<b face='monospace'>concat: 3.141<u>underline</u></b>",
	},

	{
		"text with special characters",
		Tag("b", "<>&amp;'\"="),
		"<b>&lt;&gt;&amp;amp;&#39;&#34;=</b>",
	},
	{
		"text with 'valid' html",
		Tag("i", "<b color='red'>bold</b>"),
		"<i>&lt;b color=&#39;red&#39;&gt;bold&lt;/b&gt;</i>",
	},

	{
		"simple span collapsing",
		Span("string-1", " ", "string-2", " ", 2.718),
		"string-1 string-2 2.718",
	},
	{
		"span collapsing with child nodes",
		Span(Textf("%s-%d", "string", 1), Tag("u", " "), Span(Tag("b", "e="), 2.718, "..."), Span()),
		"string-1<u> </u><b>e=</b>2.718...",
	},
}

func TestStringifying(t *testing.T) {
	for _, tc := range stringifyingTests {
		assert.Equal(t, tc.expected, tc.node.Pango(), tc.desc)
	}
}

var result string
var resultNode Node

func benchmarkConstructOnly(b *testing.B, fn func() Node) {
	var r Node
	for n := 0; n < b.N; n++ {
		r = fn()
	}
	resultNode = r
}

func benchmarkConstructAndStringify(b *testing.B, fn func() Node) {
	var s string
	for n := 0; n < b.N; n++ {
		s = fn().Pango()
	}
	result = s
}

func benchmarkStringifyOnly(b *testing.B, fn func() Node) {
	var s string
	node := fn()
	for n := 0; n < b.N; n++ {
		s = node.Pango()
	}
	result = s
}

func empty() Node                          { return Span() }
func BenchmarkEmpty(b *testing.B)          { benchmarkConstructAndStringify(b, empty) }
func BenchmarkEmptyConstruct(b *testing.B) { benchmarkConstructOnly(b, empty) }
func BenchmarkEmptyStringify(b *testing.B) { benchmarkStringifyOnly(b, empty) }

func simple() Node                          { return Tag("b") }
func BenchmarkSimple(b *testing.B)          { benchmarkConstructAndStringify(b, simple) }
func BenchmarkSimpleConstruct(b *testing.B) { benchmarkConstructOnly(b, simple) }
func BenchmarkSimpleStringify(b *testing.B) { benchmarkStringifyOnly(b, simple) }

func textonly() Node                      { return Textf("%s-%d", "test", 1024) }
func BenchmarkText(b *testing.B)          { benchmarkConstructAndStringify(b, textonly) }
func BenchmarkTextConstruct(b *testing.B) { benchmarkConstructOnly(b, textonly) }
func BenchmarkTextStringify(b *testing.B) { benchmarkStringifyOnly(b, textonly) }

func complex() Node {
	return Span(
		Font("monospace"), Weight(600), Rise(400), Size(14.0),
		Tag("b", Font("serif"), Textf("Number %d", 42)),
		Tag("small", Tag("u", "small underline")),
		Tag("i", Tag("b", Tag("u", Tag("small", Textf("%s %s %s!", "all", "the", "tags"))))),
	)
}
func BenchmarkComplex(b *testing.B)          { benchmarkConstructAndStringify(b, complex) }
func BenchmarkComplexConstruct(b *testing.B) { benchmarkConstructOnly(b, complex) }
func BenchmarkComplexStringify(b *testing.B) { benchmarkStringifyOnly(b, complex) }
