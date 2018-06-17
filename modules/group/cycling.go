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

// Cyclic is a group that supports cyclic between its modules.
// It provides an interface to move to the next, previous, or
// directly indexed module.
type Cyclic interface {
	Group

	// Visible returns the index of the currently active module.
	Visible() int

	// Previous switches to the previous module.
	Previous()

	// Next switches to the next module.
	Next()

	// Show sets the currently active module.
	Show(int)

	// Count returns the number of modules in this group
	Count() int

	// Button returns a button with the given output that switches
	// to the next module in the group when clicked.
	Button(bar.Output) Button
}

// Cycling returns a new cyclic group.
func Cycling() Cyclic {
	return &cyclic{}
}

// cyclic implements the Cyclic group. It stores a list
// of modules and the index of the currently visible one.
type cyclic struct {
	sync.Mutex
	modules []*module
	current int
}

// Add adds a module to the cyclic group. The returned module
// will not output anything unless it's currently active.
func (g *cyclic) Add(original bar.Module) WrappedModule {
	g.Lock()
	defer g.Unlock()
	index := len(g.modules)
	m := newWrappedModule(original, g.current == index)
	l.Attachf(g, m, "[%d]", len(g.modules))
	l.Label(m, l.ID(m.Module))
	l.Attach(m, m.Module, "")
	g.modules = append(g.modules, m)
	return m
}

func (g *cyclic) Visible() int {
	g.Lock()
	defer g.Unlock()
	return g.current
}

func (g *cyclic) Previous() {
	g.Show(g.Visible() - 1)
}

func (g *cyclic) Next() {
	g.Show(g.Visible() + 1)
}

func (g *cyclic) Show(index int) {
	l.Log("%s: show %d", l.ID(g), index)
	count := g.Count()
	if count == 0 {
		index = 0
	} else {
		// Handle wrap around on either side.
		index = (index + count) % count
	}
	g.Lock()
	defer g.Unlock()
	for idx, m := range g.modules {
		m.SetVisible(idx == index)
	}
	g.current = index
}

func (g *cyclic) Count() int {
	g.Lock()
	defer g.Unlock()
	return len(g.modules)
}

func (g *cyclic) Button(output bar.Output) Button {
	b := &button{}
	b.Set(output)
	b.OnClick(func(e bar.Event) {
		switch e.Button {
		case bar.ButtonLeft, bar.ScrollDown, bar.ScrollRight, bar.ButtonForward:
			g.Next()
		case bar.ButtonRight, bar.ScrollUp, bar.ScrollLeft, bar.ButtonBack:
			g.Previous()
		}
	})
	return b
}
