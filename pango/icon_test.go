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

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/testing/pango"
)

func TestNoProviders(t *testing.T) {
	iconProviders = map[string]IconProvider{}
	assert.Empty(t,
		Icon("anything-iconname").Pango(),
		"when no providers are added")

	assert.Empty(t,
		Icon("alert").Pango(),
		"when no providers are added")
}

type singleIconProvider string

func (s singleIconProvider) Icon(name string) *Node {
	if string(s) == name {
		return Textf("s:%s", s).Small()
	}
	return nil
}

func TestProviders(t *testing.T) {
	iconProviders = map[string]IconProvider{}
	AddIconProvider("t1", singleIconProvider("foo"))
	AddIconProvider("t2", singleIconProvider("bar"))

	assert.Empty(t, Icon("t0-bar").Pango(), "non-existent provider")
	assert.Empty(t, Icon("t1-bar").Pango(), "non-existent icon")
	pango.AssertText(t, "s:bar", Icon("t2-bar").Pango(),
		"provider name is not passed to Icon(...)")
}
