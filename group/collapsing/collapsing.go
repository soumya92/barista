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
package collapsing // import "barista.run/group/collapsing"

import (
	"sync"
	"sync/atomic"

	"barista.run/bar"
	"barista.run/base/click"
	"barista.run/base/notifier"
	"barista.run/group"
	l "barista.run/logging"
	"barista.run/outputs"
)

// ButtonFunc produces outputs for buttons in a collapsing group.
type ButtonFunc func(Controller) (start, end bar.Output)

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
	expanded   atomic.Value // of bool
	buttonFunc ButtonFunc

	sync.Mutex
	notifyCh <-chan struct{}
	notifyFn func()
}

// Group returns a new collapsing group, and a linked controller.
func Group(m ...bar.Module) (bar.Module, Controller) {
	g := &grouper{buttonFunc: DefaultButtons}
	g.expanded.Store(false)
	g.notifyFn, g.notifyCh = notifier.New()
	return group.New(g, m...), g
}

// DefaultButtons returns the default button outputs:
// - When expanded, a '>' and '<' on either side.
// - When collapsed, a single '+'.
func DefaultButtons(c Controller) (start, end bar.Output) {
	if c.Expanded() {
		return outputs.Text(">").OnClick(click.Left(c.Collapse)),
			outputs.Text("<").OnClick(click.Left(c.Collapse))
	}
	return outputs.Text("+").OnClick(click.Left(c.Expand)), nil
}

func (g *grouper) Visible(int) bool {
	return g.Expanded()
}

func (g *grouper) Buttons() (start, end bar.Output) {
	return g.buttonFunc(g)
}

func (g *grouper) Signal() <-chan struct{} {
	return g.notifyCh
}

func (g *grouper) Expanded() bool {
	return g.expanded.Load().(bool)
}

func (g *grouper) Collapse() {
	g.setExpanded(false)
}

func (g *grouper) Expand() {
	g.setExpanded(true)
}

func (g *grouper) Toggle() {
	g.setExpanded(!g.Expanded())
}

func (g *grouper) setExpanded(expanded bool) {
	// Group calls Visible once for each module. To ensure a consistent value
	// across the entire set, we prevent changes to expanded while the lock is
	// held. Group only releases the lock once it's done with the grouper.
	g.Lock()
	defer g.Unlock()
	if g.Expanded() == expanded {
		return
	}
	l.Fine("%s.expanded = %v", l.ID(g), expanded)
	g.expanded.Store(expanded)
	g.notifyFn()
}

func (g *grouper) ButtonFunc(f ButtonFunc) {
	g.Lock()
	defer g.Unlock()
	g.buttonFunc = f
	g.notifyFn()
}
