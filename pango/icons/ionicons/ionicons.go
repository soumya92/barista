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
	"encoding/json"
	"io"
	"io/ioutil"
	"strings"

	"github.com/soumya92/barista/pango/icons"
)

type ioniconsConfig struct {
	Icons []struct {
		Name string `json:"name"`
		Code string `json:"code"`
	} `json:"icons"`
}

// LoadMd initialises the ionicons icon provider from the given repo.
// It loads the material design icons by default, but the iOS style icons
// are available with the ios- prefix.
// pango.Icon("ion-alarm") will load the material version, while
// pango.Icon("ion-ios-alarm") will load the ios version.
func LoadMd(repoPath string) error {
	return loadWithDefaultPrefix(repoPath, "md-")
}

// LoadIos initialises the ionicons icon provider from the given repo.
// It loads the iOS icons by default, but the material design style icons
// are available with the md- prefix.
// pango.Icon("ion-alarm") will load the iOS version, while
// pango.Icon("ion-md-alarm") will load the material version.
func LoadIos(repoPath string) error {
	return loadWithDefaultPrefix(repoPath, "ios-")
}

// Load initialises the ionicons icon provider from the given repo.
// It does not strip any prefix, so both iOS and material design icons
// must be prefixed before use (e.g. "ion-ios-alarm" and "ion-md-alarm").
func Load(repoPath string) error {
	return loadWithDefaultPrefix(repoPath, "")
}

func loadWithDefaultPrefix(repoPath string, defaultPrefix string) error {
	return icons.NewProvider("ion", icons.Config{
		RepoPath: repoPath,
		FilePath: "scripts/manifest.json",
		Font:     "Ionicons",
	}).LoadFromFile(func(f io.Reader, add func(string, string)) error {
		manifest, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		conf := ioniconsConfig{}
		err = json.Unmarshal(manifest, &conf)
		if err != nil {
			return err
		}
		for _, icon := range conf.Icons {
			value := strings.TrimPrefix(icon.Code, "0x")
			symbol, err := icons.SymbolFromHex(value)
			if err != nil {
				return err
			}
			add(strings.TrimPrefix(icon.Name, defaultPrefix), symbol)
		}
		return nil
	})
}
