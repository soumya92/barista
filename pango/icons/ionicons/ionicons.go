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
Package ionicons provides support for Ionicicons
from https://github.com/driftyco/ionicons

It uses scss/_ionicons-variables.scss to get the list of icons,
and requires fonts/ionicons.ttf to be installed.
*/
package ionicons

import (
	"strings"
	"unicode"

	"github.com/soumya92/barista/pango/icons"
)

// Load initialises the ionicons icon provider from the given repo.
func Load(repoPath string) error {
	return icons.NewProvider("ion", icons.Config{
		RepoPath: repoPath,
		FilePath: "scss/_ionicons-variables.scss",
		Font:     "Ionicons",
	}).LoadByLines(func(line string, add func(string, string)) error {
		colon := strings.Index(line, ":")
		if colon < 0 {
			return nil
		}
		name := line[:colon]
		if !strings.HasPrefix(name, "$ionicon-var-") {
			return nil
		}
		name = strings.TrimPrefix(name, "$ionicon-var-")
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
}
