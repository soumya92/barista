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
Package group provides a module that "groups" existing modules, and allows
expanding/collapsing the group as a whole, or cycling through grouped module.

To group modules, construct a new group instance, and wrap each module
when adding it to the bar. Then add the Button() to add the default button,
which is toggle for collapsing, and next for cycling. Or add a static text
module and use the click handler to get more fined grain control over the
group.

 g := group.Collapsing()
 bar.Run(
   g.Add(localtime.New(...)),
   g.Add(shell.Every(...)),
   g.Button(outputs.Text("+"), outputs.Text("-")),
 )
*/
package group

import (
	"sync"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
)

// Group is the common interface for all kinds of module "groups".
// It supports adding a module and returning a wrapped module that
// behaves according to the group's rules.
type Group interface {
	Add(bar.Module) WrappedModule
}

// Button is a bar module that supports being clicked.
type Button interface {
	bar.Module
	bar.Clickable
}

// button implements Button and provides additional methods that
// groups can use to control its behaviour.
type button struct {
	base.SimpleClickHandler
	base.Value // of bar.Output
}

func (b *button) Stream(s bar.Sink) {
	sub := b.Subscribe()
	for {
		s.Output(b.Get().(bar.Output))
		<-sub
	}
}

// module wraps a bar.Module with a "visibility" modifier, and only
// sends output when it's visible. Otherwise it outputs nothing.
type module struct {
	bar.Module
	bar.Sink
	sync.Mutex
	restarted  chan bool
	lastOutput bar.Output
	visible    bool
	finished   bool
}

// WrappedModule implements bar.Module, Clickable, and Pausable.
// It forwards calls to the wrapped module only when supported.
type WrappedModule interface {
	bar.Module
	bar.Clickable
}

func newWrappedModule(m bar.Module, visible bool) *module {
	return &module{
		Module:    m,
		restarted: make(chan bool),
		visible:   visible,
	}
}

// Stream sets up the output pipeline to filter outputs when hidden.
func (m *module) Stream(s bar.Sink) {
	m.Lock()
	m.Sink = s
	m.Unlock()
	wSink := wrappedSink(m, s)
	for {
		m.Module.Stream(wSink)
		m.Lock()
		m.finished = true
		m.Unlock()
		<-m.restarted
	}
}

// Click passes through the click event if supported by the wrapped module.
func (m *module) Click(e bar.Event) {
	m.Lock()
	if m.finished && isRestartableClick(e) {
		l.Log("%s restarted by wrapper", l.ID(m.Module))
		m.finished = false
		m.restarted <- true
		m.Unlock()
		return
	}
	m.Unlock()
	if clickable, ok := m.Module.(bar.Clickable); ok {
		clickable.Click(e)
	}
}

// isRestartableClick mimics the function in barista main.
// Modules that have finished can still be hidden/shown,
// so the wrapped module cannot close the channel.
func isRestartableClick(e bar.Event) bool {
	return e.Button == bar.ButtonLeft ||
		e.Button == bar.ButtonRight ||
		e.Button == bar.ButtonMiddle
}

// SetVisible sets the module visibility and updates the output accordingly.
func (m *module) SetVisible(visible bool) {
	m.Lock()
	defer m.Unlock()
	if m.visible == visible {
		return
	}
	l.Fine("%s: visible %v", l.ID(m), visible)
	m.visible = visible
	if visible {
		m.Output(m.lastOutput)
	} else {
		m.Output(nil)
	}
}

func wrappedSink(m *module, s bar.Sink) bar.Sink {
	return func(o bar.Output) {
		m.Lock()
		m.lastOutput = o
		visible := m.visible
		m.Unlock()
		if visible {
			s.Output(o)
		}
	}
}
