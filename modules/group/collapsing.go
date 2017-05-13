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

package group

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
)

// Collapsable is a group that supports expanding/collapsable.
// When expanded (default state), all modules are visible,
// when collapsed, no modules are visible.
type Collapsable interface {
	Group
	Collapsed() bool
	Collapse()
	Expand()
	Toggle()
	Button(bar.Output, bar.Output) Button
}

// Collapsing returns a new collapsable group.
func Collapsing() Collapsable {
	return &collapsable{}
}

// collapsable implements the Collapsable group. It stores a list
// of modules and whether it's expanded or collapsed.
type collapsable struct {
	modules   []*module
	collapsed bool
}

// Add adds a module to the collapsable group. The returned module
// will not output anything when the group is collapsed.
func (g *collapsable) Add(original bar.Module) bar.Module {
	m := &module{
		Module:  original,
		channel: make(chan bar.Output),
		visible: !g.collapsed,
	}
	g.modules = append(g.modules, m)
	return m
}

// Collapsed returns true if the group is collapsed.
func (g *collapsable) Collapsed() bool {
	return g.collapsed
}

// Collapse collapses the group and hides all modules.
func (g *collapsable) Collapse() {
	g.collapsed = true
	g.syncState()
}

// Expand expands the group and shows all modules.
func (g *collapsable) Expand() {
	g.collapsed = false
	g.syncState()
}

// Toggle toggles the visibility of all modules.
func (g *collapsable) Toggle() {
	g.collapsed = !g.collapsed
	g.syncState()
}

// Button returns a button with the given output for the
// collapsed and expanded states respectively that toggles
// the group when clicked.
func (g *collapsable) Button(collapsed, expanded bar.Output) Button {
	outputFunc := func() bar.Output {
		if g.collapsed {
			return collapsed
		}
		return expanded
	}
	b := base.New()
	b.Output(outputFunc())
	b.OnClick(func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			g.Toggle()
			b.Output(outputFunc())
		}
	})
	return b
}

// syncState syncs the visible state of all modules.
func (g *collapsable) syncState() {
	for _, m := range g.modules {
		m.SetVisible(!g.collapsed)
	}
}
