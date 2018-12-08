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

It uses metadata/icons.yml to get the list of icons,
and requires fonts/fontawesome-webfont.ttf to be installed.
*/
package fontawesome // import "barista.run/pango/icons/fontawesome"

import (
	"fmt"
	"path/filepath"

	"barista.run/pango"
	"barista.run/pango/icons"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

type faMetadata struct {
	Code   string   `yaml:"unicode"`
	Styles []string `yaml:"styles"`
}

var fs = afero.NewOsFs()

// Load initialises the fontawesome icon provider from the given repo.
func Load(repoPath string) error {
	f, err := fs.Open(filepath.Join(repoPath, "metadata/icons.yml"))
	if err != nil {
		return err
	}
	defer f.Close()

	// Defaults to solid since that style has the most icons available.
	faSolid := icons.NewProvider("fa")
	faSolid.Font("Font Awesome 5 Free")
	faSolid.AddStyle(func(n *pango.Node) { n.Weight(900) })

	faBrands := icons.NewProvider("fab")
	faBrands.Font("Font Awesome 5 Brands")

	faRegular := icons.NewProvider("far")
	faRegular.Font("Font Awesome 5 Free")

	styles := map[string]*icons.Provider{
		"solid":   faSolid,
		"regular": faRegular,
		"brands":  faBrands,
	}

	var glyphs map[string]faMetadata
	err = yaml.NewDecoder(f).Decode(&glyphs)
	if err != nil {
		return err
	}
	for name, meta := range glyphs {
		for _, style := range meta.Styles {
			p, ok := styles[style]
			if !ok {
				return fmt.Errorf("Unknown FontAwesome style: '%s'", style)
			}
			err = p.Hex(name, meta.Code)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
