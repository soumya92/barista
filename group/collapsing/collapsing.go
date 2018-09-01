// Copyright 2018 Google Inc.
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
Package collapsing provides a group that supports expanding/collapsing,
and a controller to allow programmatic expansion/collapse.

When collapsed (default state), only a button to expand is visible.
When expanded, all module outputs are shown, and buttons to collapse.
*/
package collapsing

import (
	"sync"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/notifier"
	"github.com/soumya92/barista/group"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// ButtonFunc produces outputs for buttons in a collapsing group.
type ButtonFunc func(expanded bool) (start, end bar.Output)

// Controller provides an interface to control a collapsing group.
type Controller interface {
	// Expanded returns true if the group is expanded (showing output).
	Expanded() bool
	// Collapse collapses the group and hides all modules.
	Collapse()
	// Expand expands the group and shows all modules.
	Expand()
	// Toggle toggles the visibility of all modules.
	Toggle()
	// ButtonFunc controls the output for the button(s).
	ButtonFunc(ButtonFunc)
}

// grouper implements a collapsing grouper.
type grouper struct {
	expanded   bool
	buttonFunc ButtonFunc

	sync.Mutex
	notifyCh <-chan struct{}
	notifyFn func()
}

// Group returns a new collapsing group, and a linked controller.
func Group(m ...bar.Module) (bar.Module, Controller) {
	g := &grouper{buttonFunc: DefaultButtons}
	g.notifyFn, g.notifyCh = notifier.New()
	return group.New(g, m...), g
}

// DefaultButtons returns the default button outputs:
// - When expanded, a '>' and '<' on either side.
// - When collapsed, a single '+'.
func DefaultButtons(expanded bool) (start, end bar.Output) {
	if expanded {
		return outputs.Text(">"), outputs.Text("<")
	}
	return outputs.Text("+"), nil
}

func (g *grouper) Visible(int) bool {
	return g.expanded
}

func (g *grouper) Buttons() (start, end bar.Output) {
	onClick := func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			g.Toggle()
		}
	}
	start, end = g.buttonFunc(g.expanded)
	return outputs.Group(start).OnClick(onClick),
		outputs.Group(end).OnClick(onClick)
}

func (g *grouper) Signal() <-chan struct{} {
	return g.notifyCh
}

func (g *grouper) Expanded() bool {
	g.Lock()
	defer g.Unlock()
	return g.expanded
}

func (g *grouper) Collapse() {
	g.Lock()
	defer g.Unlock()
	g.setExpanded(false)
}

func (g *grouper) Expand() {
	g.Lock()
	defer g.Unlock()
	g.setExpanded(true)
}

func (g *grouper) Toggle() {
	g.Lock()
	defer g.Unlock()
	g.setExpanded(!g.expanded)
}

func (g *grouper) setExpanded(expanded bool) {
	if g.expanded == expanded {
		return
	}
	l.Fine("%s.expanded = %v", l.ID(g), expanded)
	g.expanded = expanded
	g.notifyFn()
}

func (g *grouper) ButtonFunc(f ButtonFunc) {
	g.Lock()
	defer g.Unlock()
	g.buttonFunc = f
	g.notifyFn()
}
