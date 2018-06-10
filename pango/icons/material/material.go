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
Package material provides support for Google's Material Design Icons
from https://github.com/google/material-design-icons

It uses iconfont/codepoints to get the list of icons,
and requires iconfont/MaterialIcons-Regular.ttf to be installed.
*/
package material

import (
	"fmt"
	"strings"

	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/pango/icons"
)

// Load initialises the material design icon provider from the given repo.
func Load(repoPath string) error {
	return icons.NewProvider("material", icons.Config{
		RepoPath: repoPath,
		FilePath: "iconfont/codepoints",
		Font:     "Material Icons",
		Styler:   func(n *pango.Node) { n.UltraLight().Rise(-1600) },
	}).LoadByLines(func(line string, add func(string, string)) error {
		components := strings.Split(line, " ")
		if len(components) != 2 {
			return fmt.Errorf("Unexpected line in 'iconfont/codepoints'")
		}
		symbol, err := icons.SymbolFromHex(components[1])
		if err != nil {
			return err
		}
		// Material Design Icons uses '_', but all other fonts use '-',
		// so we'll normalise it here.
		name := strings.Replace(components[0], "_", "-", -1)
		add(name, symbol)
		return nil
	})
}
