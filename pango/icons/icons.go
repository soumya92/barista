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
  return pango.Span(
    material.Icon("today", colors.Hex("#ddd")),
    now.Sprintf("%H:%M"),
  )
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
	symbols map[string]string
	attrs   []pango.Attribute
}

// Icon creates a pango span that renders the named icon.
// It looks up the name in the loaded mapping, and if found,
// merges the default styles with the user provided styles (if any)
// to produce a <span> that will render the requested icon.
func (p *Provider) Icon(name string, style ...pango.Attribute) pango.Node {
	if p == nil {
		return pango.Span()
	}
	symbol, ok := p.symbols[name]
	if !ok {
		return pango.Span()
	}
	things := []interface{}{symbol}
	overrides := make(map[string]bool)
	for _, attr := range style {
		things = append(things, attr)
		overrides[attr.AttrName()] = true
	}
	for _, attr := range p.attrs {
		if !overrides[attr.AttrName()] {
			things = append(things, attr)
		}
	}
	return pango.Span(things...)
}

// Config stores Configuration options
// for building an IconProvider.
type Config struct {
	RepoPath string
	FilePath string
	Font     string
	attrs    []pango.Attribute
}

// Styles sets any default pango styles (e.g. weight, baseline)
// that should apply to all icons. User-defined styles will override
// any styles provided here.
func (c *Config) Styles(attrs ...pango.Attribute) {
	for _, attr := range attrs {
		c.attrs = append(c.attrs, attr)
	}
}

var fs = afero.NewOsFs()

// LoadFromFile creates an IconProvider by passing to the parse
// function an io.Reader for the source file, and a function to add
// icons to the provider's map.
func (c *Config) LoadFromFile(parseFile func(io.Reader, func(string, string)) error) (*Provider, error) {
	f, err := fs.Open(filepath.Join(c.RepoPath, c.FilePath))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	i := Provider{
		symbols: make(map[string]string),
		attrs:   append(c.attrs, pango.Font(c.Font)),
	}
	err = parseFile(f, func(name, symbol string) {
		i.symbols[name] = symbol
	})
	return &i, err
}

// LoadByLines creates an IconProvider by passing to the parse
// function each line of the source file, and a function to add
// icons to the provider's map.
func (c *Config) LoadByLines(parseLine func(string, func(string, string)) error) (*Provider, error) {
	return c.LoadFromFile(func(f io.Reader, add func(string, string)) error {
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
