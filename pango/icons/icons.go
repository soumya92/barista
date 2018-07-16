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
Package icons provides an interface for using icon fonts in a bar.
To use an icon font:
 - Clone a supported repository
 - Link the ttf into ~/.fonts
 - Load the icon by passing it the path to the repo
 - Use icons as pango constructs in your bar

Compatible icon fonts:
 - Material Design Icons (+community fork)
 - FontAwesome
 - Ionicons
 - Typicons

Example usage:
  material.Load("/Users/me/Github/google/material-design-icons")
  ...
  return pango.Icon("material-today").Color(colors.Hex("#ddd")).
      Append(pango.Text(now.Sprintf("%H:%M")))
*/
package icons

import (
	"bufio"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/afero"

	"github.com/soumya92/barista/pango"
)

// Provider provides pango nodes for icons
type Provider struct {
	font    string
	symbols map[string]string
	styler  func(*pango.Node)
	file    string
}

// Icon creates a pango node that renders the named icon.
func (p *Provider) Icon(name string) *pango.Node {
	symbol, ok := p.symbols[name]
	if !ok {
		return nil
	}
	n := pango.Text(symbol).Font(p.font)
	if p.styler != nil {
		p.styler(n)
	}
	return n
}

// NewProvider creates a new icon provider with the given name,
// registers it with pango.Icon, and returns it so that an appropriate
// Load method can be used.
func NewProvider(name string, c Config) *Provider {
	p := &Provider{
		font:    c.Font,
		symbols: map[string]string{},
		styler:  c.Styler,
		file:    filepath.Join(c.RepoPath, c.FilePath),
	}
	pango.AddIconProvider(name, p)
	return p
}

// Config stores Configuration options for building an IconProvider.
type Config struct {
	// Path to a git repository, typically supplied by the user.
	RepoPath string
	// Path to the file within the repository.
	FilePath string
	// Name of the font face.
	Font string
	// An optional function that adds any required pango styling,
	// in addition to the font. (e.g. light/ultralight weight)
	Styler func(*pango.Node)
}

var fs = afero.NewOsFs()

// LoadFromFile creates an IconProvider by passing to the parse
// function an io.Reader for the source file, and a function to add
// icons to the provider's map.
func (p *Provider) LoadFromFile(parseFile func(io.Reader, func(string, string)) error) error {
	f, err := fs.Open(p.file)
	if err != nil {
		return err
	}
	defer f.Close()
	return parseFile(f, func(name, symbol string) {
		p.symbols[name] = symbol
	})
}

// LoadByLines creates an IconProvider by passing to the parse
// function each line of the source file, and a function to add
// icons to the provider's map.
func (p *Provider) LoadByLines(parseLine func(string, func(string, string)) error) error {
	return p.LoadFromFile(func(f io.Reader, add func(string, string)) error {
		s := bufio.NewScanner(f)
		s.Split(bufio.ScanLines)
		for s.Scan() {
			line := strings.TrimSpace(s.Text())
			if line == "" {
				continue
			}
			err := parseLine(line, add)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// SymbolFromHex parses a hex string (e.g. "1F44D") and converts
// it to a string (e.g. "👍").
func SymbolFromHex(hex string) (string, error) {
	intVal, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return "", err
	}
	return string(intVal), nil
}
