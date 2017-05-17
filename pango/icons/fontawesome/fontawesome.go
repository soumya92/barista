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

/*
Package fontawesome provides support for FontAwesome Icons
from https://github.com/FortAwesome/Font-Awesome

It uses scss/_variables.scss to get the list of icons,
and requires fonts/fontawesome-webfont.ttf to be installed.
*/
package fontawesome

import (
	"strings"
	"unicode"

	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/pango/icons"
)

var provider *icons.Provider

// Icon returns a pango node for the given icon name and styles.
func Icon(name string, style ...pango.Attribute) pango.Node {
	return provider.Icon(name, style...)
}

// Load initialises the fontawesome icon provider from the given repo.
func Load(repoPath string) error {
	c := icons.Config{
		RepoPath: repoPath,
		FilePath: "scss/_variables.scss",
		Font:     "FontAwesome",
	}
	var err error
	provider, err = c.LoadByLines(func(line string, add func(string, string)) error {
		colon := strings.Index(line, ":")
		if colon < 0 {
			return nil
		}
		name := line[:colon]
		if !strings.HasPrefix(name, "$fa-var-") {
			return nil
		}
		name = strings.TrimPrefix(name, "$fa-var-")
		value := strings.TrimFunc(line[colon+1:], func(r rune) bool {
			return unicode.IsSpace(r) || r == '"' || r == '\\' || r == ';'
		})
		sym, err := icons.SymbolFromHex(value)
		if err != nil {
			return err
		}
		add(name, sym)
		return nil
	})
	return err
}
