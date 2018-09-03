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
	"testing"

	"barista.run/testing/pango"

	"github.com/stretchr/testify/require"
)

func TestNoProviders(t *testing.T) {
	iconProviders = map[string]IconProvider{}
	require.Empty(t,
		Icon("anything-iconname").String(),
		"when no providers are added")

	require.Empty(t,
		Icon("alert").String(),
		"when no providers are added")
}

func singleIconProvider(name string) IconProvider {
	return func(s string) *Node {
		if string(s) == name {
			return Textf("s:%s", s).Small()
		}
		return nil
	}
}

func TestProviders(t *testing.T) {
	iconProviders = map[string]IconProvider{}
	AddIconProvider("t1", singleIconProvider("foo"))
	AddIconProvider("t2", singleIconProvider("bar"))

	require.Empty(t, Icon("t0-bar").String(), "non-existent provider")
	require.Empty(t, Icon("t1-bar").String(), "non-existent icon")
	pango.AssertText(t, "s:bar", Icon("t2-bar").String(),
		"provider name is not passed to Icon(...)")
}
