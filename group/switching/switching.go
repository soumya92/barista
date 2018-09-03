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
package switching // import "barista.run/group/switching"

import (
	"sync"

	"barista.run/bar"
	"barista.run/base/notifier"
	"barista.run/group"
	l "barista.run/logging"
	"barista.run/outputs"
)

// ButtonFunc produces outputs for buttons in a switching group.
type ButtonFunc func(current, len int) (start, end bar.Output)

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
	current    int
	count      int
	buttonFunc ButtonFunc

	sync.Mutex
	notifyCh <-chan struct{}
	notifyFn func()
}

// Group returns a new switching group, and a linked controller.
func Group(m ...bar.Module) (bar.Module, Controller) {
	g := &grouper{count: len(m), buttonFunc: DefaultButtons}
	g.notifyFn, g.notifyCh = notifier.New()
	return group.New(g, m...), g
}

// DefaultButtons provides the default switching buttons:
// - '<' at the start, if there are modules before the current one,
// - '>' at the end, if there are modules after the current one.
func DefaultButtons(current, count int) (start, end bar.Output) {
	if current > 0 {
		start = outputs.Textf("<")
	}
	if current+1 < count {
		end = outputs.Textf(">")
	}
	return start, end
}

func (g *grouper) Visible(idx int) bool {
	return g.current == idx
}

func (g *grouper) Buttons() (start, end bar.Output) {
	start, end = g.buttonFunc(g.current, g.count)
	startClick := func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			g.Previous()
		}
	}
	endClick := func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			g.Next()
		}
	}
	return outputs.Group(start).OnClick(startClick),
		outputs.Group(end).OnClick(endClick)
}

func (g *grouper) Signal() <-chan struct{} {
	return g.notifyCh
}

func (g *grouper) Current() int {
	g.Lock()
	defer g.Unlock()
	return g.current
}

func (g *grouper) Previous() {
	g.Lock()
	defer g.Unlock()
	g.setIndex(g.current - 1)
}

func (g *grouper) Next() {
	g.Lock()
	defer g.Unlock()
	g.setIndex(g.current + 1)
}

func (g *grouper) Show(index int) {
	g.Lock()
	defer g.Unlock()
	g.setIndex(index)
}

func (g *grouper) Count() int {
	g.Lock()
	defer g.Unlock()
	return g.count
}

func (g *grouper) setIndex(index int) {
	// Handle wrap around on either side.
	g.current = (index + g.count) % g.count
	l.Fine("%s switched to #%d", l.ID(g), g.current)
	g.notifyFn()
}

func (g *grouper) ButtonFunc(f ButtonFunc) {
	g.Lock()
	defer g.Unlock()
	g.buttonFunc = f
	g.notifyFn()
}
