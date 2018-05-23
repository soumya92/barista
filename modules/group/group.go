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
	"github.com/soumya92/barista/outputs"
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
	base.Channel
}

func (b *button) Stream() <-chan bar.Output {
	return b.Channel
}

func newButton() *button {
	return &button{Channel: base.NewChannel()}
}

// module wraps a bar.Module with a "visibility" modifier, and only
// sends output when it's visible. Otherwise it outputs nothing.
type module struct {
	bar.Module
	sync.Mutex
	channel    chan bar.Output
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

// Stream sets up the output pipeline to filter outputs when hidden.
func (m *module) Stream() <-chan bar.Output {
	m.Lock()
	m.channel = make(chan bar.Output, 10)
	go m.pipeWhenVisible(m.Module.Stream(), m.channel)
	m.Unlock()
	return m.channel
}

// Click passes through the click event if supported by the wrapped module.
func (m *module) Click(e bar.Event) {
	m.Lock()
	if m.finished && isRestartableClick(e) {
		l.Log("%s restarted by wrapper", l.ID(m.Module))
		go m.pipeWhenVisible(m.Module.Stream(), m.channel)
		m.finished = false
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
		m.channel <- m.lastOutput
	} else {
		m.channel <- outputs.Empty()
	}
}

func (m *module) pipeWhenVisible(input <-chan bar.Output, output chan<- bar.Output) {
	for out := range input {
		m.Lock()
		m.lastOutput = out
		visible := m.visible
		m.Unlock()
		if visible {
			output <- out
		}
	}
	m.Lock()
	m.finished = true
	m.Unlock()
}
