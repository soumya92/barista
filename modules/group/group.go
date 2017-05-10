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
- expanding/collapsing the group as a whole, or
- cycling through grouped module.

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
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
)

// Group is the common interface for all kinds of module "groups".
// It supports adding a module and returning a wrapped module that
// behaves according to the group's rules.
type Group interface {
	Add(bar.Module) bar.Module
}

// Button is a bar module that supports being clicked.
type Button interface {
	bar.Module
	bar.Clickable
}

// module wraps a bar.Module with a "visibility" modifier, and only
// sends output when it's visible. Otherwise it outputs nothing.
type module struct {
	bar.Module
	channel    chan *bar.Output
	lastOutput *bar.Output
	visible    bool
}

// Stream sets up the output pipeline to filter outputs when hidden.
func (m *module) Stream() <-chan *bar.Output {
	go m.pipeWhenVisible(m.Module.Stream(), m.channel)
	return m.channel
}

// Click passes through the click event if supported by the wrapped module.
func (m *module) Click(e bar.Event) {
	if clickable, ok := m.Module.(bar.Clickable); ok {
		clickable.Click(e)
	}
}

// Pause passes through the pause event if supported by the wrapped module.
func (m *module) Pause() {
	if pausable, ok := m.Module.(bar.Pausable); ok {
		pausable.Pause()
	}
}

// Resume passes through the resume event if supported by the wrapped module.
func (m *module) Resume() {
	if pausable, ok := m.Module.(bar.Pausable); ok {
		pausable.Resume()
	}
}

// SetVisible sets the module visibility and updates the output accordingly.
func (m *module) SetVisible(visible bool) {
	m.visible = visible
	if visible {
		m.channel <- m.lastOutput
	} else {
		m.channel <- outputs.Empty()
	}
}

func (m *module) pipeWhenVisible(input <-chan *bar.Output, output chan<- *bar.Output) {
	for out := range input {
		m.lastOutput = out
		if m.visible {
			output <- out
		}
	}
}
