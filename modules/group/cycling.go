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
	modules []*module
	current int
}

// Add adds a module to the cyclic group. The returned module
// will not output anything unless it's currently active.
func (g *cyclic) Add(original bar.Module) bar.Module {
	index := len(g.modules)
	m := &module{
		Module:  original,
		channel: make(chan bar.Output),
		visible: g.current == index,
	}
	g.modules = append(g.modules, m)
	return m
}

func (g *cyclic) Visible() int {
	return g.current
}

func (g *cyclic) Previous() {
	g.Show(g.Visible() - 1)
}

func (g *cyclic) Next() {
	g.Show(g.Visible() + 1)
}

func (g *cyclic) Show(index int) {
	// Handle wrap around on either side.
	index = (index + len(g.modules)) % len(g.modules)
	for idx, m := range g.modules {
		m.SetVisible(idx == index)
	}
	g.current = index
}

func (g *cyclic) Count() int {
	return len(g.modules)
}

func (g *cyclic) Button(output bar.Output) Button {
	b := base.New()
	b.Output(output)
	b.OnClick(func(e bar.Event) {
		switch e.Button {
		case bar.ButtonLeft, bar.ScrollDown, bar.ScrollLeft, bar.ButtonBack:
			g.Previous()
		case bar.ButtonRight, bar.ScrollUp, bar.ScrollRight, bar.ButtonForward:
			g.Next()
		}
	})
	return b
}
