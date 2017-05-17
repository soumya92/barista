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
package typicons

import (
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/pango/icons"
)

var provider *icons.Provider

// Icon returns a pango node for the given icon name and styles.
func Icon(name string, style ...pango.Attribute) pango.Node {
	return provider.Icon(name, style...)
}

type typiconsConfig struct {
	Glyphs []struct {
		Name string `yaml:"css"`
		Code string `yaml:"code"`
	} `yaml:"glyphs"`
}

// Load initialises the typicons icon provider from the given repo.
func Load(repoPath string) error {
	c := icons.Config{
		RepoPath: repoPath,
		FilePath: "config.yml",
		Font:     "Typicons",
	}
	var err error
	provider, err = c.LoadFromFile(func(f io.Reader, add func(string, string)) error {
		yml, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		conf := typiconsConfig{}
		err = yaml.Unmarshal(yml, &conf)
		if err != nil {
			return err
		}
		for _, glyph := range conf.Glyphs {
			value := strings.TrimPrefix(glyph.Code, "0x")
			symbol, err := icons.SymbolFromHex(value)
			if err != nil {
				return err
			}
			add(glyph.Name, symbol)
		}
		return nil
	})
	return err
}
