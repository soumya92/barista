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

/*
Package pango provides provides a method to test markup equality.
It compares to strings that represent pango markup while ignoring
differences in attribute order, escaping, etc.
*/
package pango

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// AssertEqual asserts that the given strings represent equivalent pango markup,
// i.e. the result of their rendering will be the same.
func AssertEqual(t *testing.T, expected, actual string, args ...interface{}) {
	expectedR, err := html.Parse(strings.NewReader(expected))
	require.NoError(t, err, args...)
	actualR, err := html.Parse(strings.NewReader(actual))
	require.NoError(t, err, args...)
	if !equalMarkup(expectedR, actualR) {
		require.Fail(t, fmt.Sprintf("%s !~= %s", expected, actual), args...)
	}
}

// AssertText asserts that the markup string has the expected text content.
// Text content ignores any tags and attributes, using only rendered text.
func AssertText(t *testing.T, expected string, markup string, args ...interface{}) {
	markupR, err := html.Parse(strings.NewReader(markup))
	require.NoError(t, err, args...)
	require.Equal(t, expected, textOf(markupR), args...)
}

func equalMarkup(a, b *html.Node) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Data != b.Data {
		return false
	}
	if len(a.Attr) != len(b.Attr) {
		return false
	}
	aAttrMap := map[string]string{}
	for _, aAttr := range a.Attr {
		aAttrMap[aAttr.Key] = aAttr.Val
	}
	for _, bAttr := range b.Attr {
		if aAttrMap[bAttr.Key] != bAttr.Val {
			return false
		}
	}
	if !equalMarkup(a.NextSibling, b.NextSibling) {
		return false
	}
	if !equalMarkup(a.FirstChild, b.FirstChild) {
		return false
	}
	return true
}

func textOf(n *html.Node) (text string) {
	if n == nil {
		return text
	}
	if n.Type == html.TextNode {
		text += n.Data
	}
	text += textOf(n.FirstChild)
	text += textOf(n.NextSibling)
	return text
}
