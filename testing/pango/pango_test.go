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

package pango

import (
	"fmt"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func TestEqual(t *testing.T) {
	cases := []struct {
		a, b string
		desc string
	}{
		{"<b>foo</b>", "<b>foo</b>", "simple"},
		{"&#60;b> foo", "&lt;b&gt; foo", "basic entities"},
		{
			"<span attr='value'>content</span>",
			"<span attr='value'>content</span>",
			"with attribute",
		},
		{
			"<span attr1='1' attr2='2'>baz</span>",
			"<span attr2='2' attr1='1'>baz</span>",
			"re-ordered attributes",
		},
		{
			"<span attr1='1' attr2='2'><b>a</b><tt><u>z</u></tt></span>",
			"<span attr2='2' attr1='1'><b>a</b><tt><u>z</u></tt></span>",
			"nested tags",
		},
		{
			"<abbr title='something'>sth</abbr>",
			`<abbr title="something">sth</abbr>`,
			"attribute quoting",
		},
		{
			"<u title='<-- this way'>look</u>",
			"<u title='&lt;-- this way'>look</u>",
			"attribute escaping",
		},
	}

	for _, tc := range cases {
		AssertEqual(t, tc.a, tc.b, tc.desc)
	}
}

func TestUnequal(t *testing.T) {
	cases := []struct {
		a, b string
		desc string
	}{
		{"<b>foo</b>", "<u>foo</u>", "tag name"},
		{"&#61;b> foo", "&lt;b&lt; foo", "basic entities"},
		{"<b>foo</b>bar", "<b>foo</b>", "truncated content"},
		{"<abbr>HTML</abbr>", "<abbr>XML</abbr>", "text content"},
		{"<u title='<-- this way'>look</u>", "<u>look</u>", "missing attribute"},
		{
			"<span attr='value'>content</span>",
			"<span attr='otherval'>content</span>",
			"attribute value",
		},
		{
			"<span attr1='value'>content</span>",
			"<span attr2='value'>content</span>",
			"attribute name",
		},
		{
			"<span attr1='1' attr2='2'>baz</span>",
			"<span attr2='1' attr1='2'>baz</span>",
			"multiple attributes",
		},
		{
			"<span attr1='1' attr2='2'><b>a</b><tt><i>z</i></tt></span>",
			"<span attr2='2' attr1='1'><b>a</b><tt>z</tt></span>",
			"nested tags",
		},
	}

	for _, tc := range cases {
		fakeT := &testing.T{}
		AssertEqual(fakeT, tc.a, tc.b)
		if !fakeT.Failed() {
			assert.Fail(t, fmt.Sprintf("Expected %s ~= %s to fail", tc.a, tc.b), tc.desc)
		}
	}
}
