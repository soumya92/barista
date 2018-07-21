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
Package mdi provides support for "Material Design Icons" from
https://materialdesignicons.com/, a fork and extension of Material.

It requires cloning the webfont repo
https://github.com/Templarian/MaterialDesign-Webfont,
uses scss/_variables.scss to get the list of icons,
and requires fonts/materialdesignicons-webfont.ttf to be installed.
*/
package mdi

import (
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/afero"

	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/pango/icons"
)

var fs = afero.NewOsFs()

// Load initialises the material design (community) icon provider
// from the given repo.
func Load(repoPath string) error {
	f, err := fs.Open(filepath.Join(repoPath, "scss/_variables.scss"))
	if err != nil {
		return err
	}
	defer f.Close()
	mdi := icons.NewProvider("mdi")
	mdi.Font("Material Design Icons")
	mdi.AddStyle(func(n *pango.Node) { n.UltraLight() })
	started := false
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if !started {
			if strings.Contains(line, "$mdi-icons:") {
				started = true
			}
			continue
		}
		if line == ");" {
			return nil
		}
		colon := strings.Index(line, ":")
		if colon < 0 {
			return fmt.Errorf("Unexpected line '%s'", line)
		}
		name := strings.TrimFunc(line[:colon], func(r rune) bool {
			return unicode.IsSpace(r) || r == '"'
		})
		value := strings.TrimFunc(line[colon+1:], func(r rune) bool {
			return unicode.IsSpace(r) || r == ','
		})
		err = mdi.Hex(name, value)
		if err != nil {
			return err
		}
	}
	if !started {
		return errors.New("Could not find any icons in _variables.scss")
	}
	return errors.New("Expected ); to end $mdi-icons, got end of file")
}
