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
 - Typicons

Example usage:
  material.Load("/Users/me/Github/google/material-design-icons")
  ...
  return pango.Icon("material-today").Color(colors.Hex("#ddd")).
      Append(pango.Text(now.Sprintf("%H:%M")))
*/
package icons // import "barista.run/pango/icons"

import (
	"strconv"

	"barista.run/pango"
)

// Provider provides pango nodes for icons
type Provider struct {
	symbols map[string]string
	styles  []func(*pango.Node)
}

// NewProvider creates a new icon provider with the given name,
// registers it with pango.Icon, and returns it so that an appropriate
// Load method can be used.
func NewProvider(name string) *Provider {
	p := &Provider{symbols: map[string]string{}}
	pango.AddIconProvider(name, p.icon)
	return p
}

// icon creates a pango node that renders the named icon.
func (p *Provider) icon(name string) *pango.Node {
	symbol, ok := p.symbols[name]
	if !ok {
		return nil
	}
	n := pango.Text(symbol)
	for _, s := range p.styles {
		s(n)
	}
	return n
}

// Hex adds a symbol to the provider where the value is given
// in hex-encoded form.
func (p *Provider) Hex(name, value string) error {
	sym, err := SymbolFromHex(value)
	if err != nil {
		return err
	}
	p.Symbol(name, sym)
	return nil
}

// Symbol adds a symbol to the provider where the value is the
// symbol/string to use for the icon.
func (p *Provider) Symbol(name, value string) {
	p.symbols[name] = value
}

// Font sets the font set on the returned pango nodes.
func (p *Provider) Font(font string) {
	p.AddStyle(func(n *pango.Node) { n.Font(font) })
}

// AddStyle sets additional styles on all returned pango nodes.
func (p *Provider) AddStyle(style func(*pango.Node)) {
	p.styles = append(p.styles, style)
}

// SymbolFromHex parses a hex string (e.g. "1F44D") and converts
// it to a string (e.g. "👍").
func SymbolFromHex(hex string) (string, error) {
	intVal, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return "", err
	}
	return string(rune(intVal)), nil
}
