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
	"sync"

	"github.com/soumya92/barista/bar"
	l "github.com/soumya92/barista/logging"
)

// Collapsable is a group that supports expanding/collapsable.
// When expanded (default state), all modules are visible,
// when collapsed, no modules are visible.
type Collapsable interface {
	Group

	// Collapsed returns true if the group is collapsed.
	Collapsed() bool

	// Collapse collapses the group and hides all modules.
	Collapse()

	// Expand expands the group and shows all modules.
	Expand()

	// Toggle toggles the visibility of all modules.
	Toggle()

	// Button returns a button with the given output for the
	// collapsed and expanded states respectively that toggles
	// the group when clicked.
	Button(collapsed, expanded bar.Output) Button
}

// Collapsing returns a new collapsable group.
func Collapsing() Collapsable {
	return &collapsable{}
}

// collapsable implements the Collapsable group. It stores a list
// of modules and whether it's expanded or collapsed.
type collapsable struct {
	sync.Mutex
	modules   []*module
	collapsed bool
}

// Add adds a module to the collapsable group. The returned module
// will not output anything when the group is collapsed.
func (g *collapsable) Add(original bar.Module) WrappedModule {
	g.Lock()
	defer g.Unlock()
	m := newWrappedModule(original, !g.collapsed)
	l.Attachf(g, m, "[%d]", len(g.modules))
	l.Label(m, l.ID(m.Module))
	l.Attach(m, m.Module, "")
	g.modules = append(g.modules, m)
	return m
}

func (g *collapsable) Collapsed() bool {
	g.Lock()
	defer g.Unlock()
	return g.collapsed
}

func (g *collapsable) Collapse() {
	g.setCollapsed(true)
}

func (g *collapsable) Expand() {
	g.setCollapsed(false)
}

func (g *collapsable) Toggle() {
	g.setCollapsed(!g.Collapsed())
}

func (g *collapsable) Button(collapsed, expanded bar.Output) Button {
	outputFunc := func() bar.Output {
		if g.Collapsed() {
			return collapsed
		}
		return expanded
	}
	b := &button{}
	b.Set(outputFunc())
	b.OnClick(func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			g.Toggle()
			b.Set(outputFunc())
		}
	})
	return b
}

func (g *collapsable) setCollapsed(collapsed bool) {
	l.Log("%s: set collapsed = %v", l.ID(g), collapsed)
	g.Lock()
	defer g.Unlock()
	g.collapsed = collapsed
	for _, m := range g.modules {
		m.SetVisible(!g.collapsed)
	}
}
