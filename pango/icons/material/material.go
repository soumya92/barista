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

It uses font/MaterialIcons-Regular.codepoints to get the list of icons,
and requires font/MaterialIcons-Regular.ttf to be installed.
*/
package material // import "barista.run/pango/icons/material"

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strings"

	"barista.run/pango"
	"barista.run/pango/icons"

	"github.com/spf13/afero"
)

var fs = afero.NewOsFs()

// Load initialises the material design icon provider from the given repo.
func Load(repoPath string) error {
	f, err := fs.Open(filepath.Join(repoPath, "font/MaterialIcons-Regular.codepoints"))
	if err != nil {
		return err
	}
	defer f.Close()
	material := icons.NewProvider("material")
	material.Font("Material Icons")
	material.AddStyle(func(n *pango.Node) { n.UltraLight().Rise(-4000) })
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		components := strings.Split(line, " ")
		if len(components) != 2 {
			return fmt.Errorf("Unexpected line '%s' in 'font/MaterialIcons-Regular.codepoints'", line)
		}
		// Material Design Icons uses '_', but all other fonts use '-',
		// so we'll normalise it here.
		name := strings.Replace(components[0], "_", "-", -1)
		value := components[1]
		err = material.Hex(name, value)
		if err != nil {
			return err
		}
	}
	return nil
}
