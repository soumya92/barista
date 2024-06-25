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

// Package switching provides a group that displays modules one at a time,
// and a controller to switch to the next/previous/specific module.
package switching

import (
	"sync"
	"sync/atomic"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/click"
	"github.com/soumya92/barista/base/notifier"
	"github.com/soumya92/barista/group"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// ButtonFunc produces outputs for buttons in a switching group.
type ButtonFunc func(Controller) (start, end bar.Output)

// Controller provides an interface to control a switching group.
type Controller interface {
	// Current returns the index of the currently active module.
	Current() int
	// Previous switches to the previous module.
	Previous()
	// Next switches to the next module.
	Next()
	// Show sets the currently active module.
	Show(int)
	// Count returns the number of modules in this group
	Count() int
	// ButtonFunc controls the output for the buttons on either end.
	ButtonFunc(ButtonFunc)
}

// grouper implements a switching grouper.
type grouper struct {
	current    atomic.Value // of int
	count      int
	buttonFunc ButtonFunc

	sync.Mutex
	notifyCh <-chan struct{}
	notifyFn func()
}

// Group returns a new switching group, and a linked controller.
func Group(m ...bar.Module) (bar.Module, Controller) {
	g := &grouper{count: len(m), buttonFunc: DefaultButtons}
	g.current.Store(0)
	g.notifyFn, g.notifyCh = notifier.New()
	return group.New(g, m...), g
}

// DefaultButtons provides the default switching buttons:
// - '<' at the start, if there are modules before the current one,
// - '>' at the end, if there are modules after the current one.
func DefaultButtons(c Controller) (start, end bar.Output) {
	if c.Current() > 0 {
		start = outputs.Textf("<").OnClick(click.Left(c.Previous))
	}
	if c.Current()+1 < c.Count() {
		end = outputs.Textf(">").OnClick(click.Left(c.Next))
	}
	return start, end
}

func (g *grouper) Visible(idx int) bool {
	return g.Current() == idx
}

func (g *grouper) Buttons() (start, end bar.Output) {
	return g.buttonFunc(g)
}

func (g *grouper) Signal() <-chan struct{} {
	return g.notifyCh
}

func (g *grouper) Current() int {
	return g.current.Load().(int)
}

func (g *grouper) Previous() {
	g.setIndex(g.Current() - 1)
}

func (g *grouper) Next() {
	g.setIndex(g.Current() + 1)
}

func (g *grouper) Show(index int) {
	g.setIndex(index)
}

func (g *grouper) Count() int {
	return g.count
}

func (g *grouper) setIndex(index int) {
	// Group calls Visible once for each module. To ensure a consistent value
	// across the entire set, we prevent changes to current while the lock is
	// held. Group only releases the lock once it's done with the grouper.
	g.Lock()
	defer g.Unlock()
	// Handle wrap around on either side.
	current := (index + g.count) % g.count
	l.Fine("%s switched to #%d", l.ID(g), current)
	g.current.Store(current)
	g.notifyFn()
}

func (g *grouper) ButtonFunc(f ButtonFunc) {
	g.Lock()
	defer g.Unlock()
	g.buttonFunc = f
	g.notifyFn()
}
