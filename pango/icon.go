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
	"strings"

	l "barista.run/logging"
)

// IconProvider is an interface for providing pango Icons.
// The function should return a pango node for the given
// icon name, or nil if an icon could not be found.
type IconProvider func(string) *Node

var iconProviders = map[string]IconProvider{}

// Icon returns a pango node that displays the given icon.
// The identifier must be of the form $provider-$name, and the returned
// node will render the $name icon using $provider. e.g. "fa-add" will
// render the "add" icon using font awesome.
func Icon(ident string) *Node {
	providerAndName := strings.SplitN(ident, "-", 2)
	if len(providerAndName) != 2 {
		l.Log("Could not identify icon provider in '%s'", ident)
		return &Node{}
	}
	provider := providerAndName[0]
	name := providerAndName[1]
	if p, ok := iconProviders[provider]; ok {
		node := p(name)
		if node != nil {
			node.attributes["fallback"] = "false"
			return New(node)
		}
	}
	return &Node{}
}

// AddIconProvider adds an icon provider for a given prefix.
// This is intended for use only by the pango/icons package.
// See "pango/icons".LoadFromFile for more details.
func AddIconProvider(name string, provider IconProvider) {
	iconProviders[name] = provider
}
