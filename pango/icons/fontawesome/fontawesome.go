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
	"bufio"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/afero"

	"github.com/soumya92/barista/pango/icons"
)

var fs = afero.NewOsFs()

// Load initialises the fontawesome icon provider from the given repo.
func Load(repoPath string) error {
	f, err := fs.Open(filepath.Join(repoPath, "web-fonts-with-css/scss/_variables.scss"))
	if err != nil {
		return err
	}
	defer f.Close()
	fa := icons.NewProvider("fa")
	fa.Font("Font Awesome 5 Free")
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		name := line[:colon]
		if !strings.HasPrefix(name, "$fa-var-") {
			continue
		}
		name = strings.TrimPrefix(name, "$fa-var-")
		value := strings.TrimFunc(line[colon+1:], func(r rune) bool {
			return unicode.IsSpace(r) || r == '"' || r == '\\' || r == ';'
		})
		err = fa.Hex(name, value)
		if err != nil {
			return err
		}
	}
	return nil
}
