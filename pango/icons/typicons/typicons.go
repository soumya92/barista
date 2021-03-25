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
Package typicons provides support for Typicons
from https://github.com/stephenhutchings/typicons.font

It uses config.yml to get the list of icons,
and requires src/font/typicons.ttf to be installed.
*/
package typicons // import "barista.run/pango/icons/typicons"

import (
	"encoding/json"
	"path/filepath"

	"barista.run/pango/icons"

	"github.com/spf13/afero"
)

var fs = afero.NewOsFs()

// Load initialises the typicons icon provider from the given repo.
func Load(repoPath string) error {
	f, err := fs.Open(filepath.Join(repoPath, "src/font/typicons.json"))
	if err != nil {
		return err
	}
	defer f.Close()
	t := icons.NewProvider("typecn")
	t.Font("Typicons")
	var conf map[string]int
	err = json.NewDecoder(f).Decode(&conf)
	if err != nil {
		return err
	}
	for name, glyphCode := range conf {
		t.Symbol(name, string(rune(glyphCode)))
	}
	return nil
}
