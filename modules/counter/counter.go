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

// Package counter demonstrates an extremely simple i3bar module that shows a counter
// which can be chnaged by clicking on it. It showcases the asynchronous nature of
// i3bar modules when written in go.
package counter // import "barista.run/modules/counter"

import (
	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
)

// Module represents a "counter" module that displays a count
// in the given format, and adjusts the count on click/scroll.
// This module exemplifies the event-based architecture of barista.
type Module struct {
	count  value.Value // of int
	format value.Value // of string
}

// New constructs a new counter module.
func New(format string) *Module {
	m := &Module{}
	l.Register(m, "format", "count")
	m.count.Set(0)
	m.format.Set(format)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	count := m.count.Get().(int)
	countSub, done := m.count.Subscribe()
	defer done()
	format := m.format.Get().(string)
	formatSub, done := m.format.Subscribe()
	defer done()
	for {
		s.Output(outputs.Textf(format, count).OnClick(m.click))
		select {
		case <-countSub:
			count = m.count.Get().(int)
		case <-formatSub:
			format = m.format.Get().(string)
		}
	}
}

// Format sets the output format.
// The given format string will receive the counter value
// as the only argument.
func (m *Module) Format(format string) *Module {
	m.format.Set(format)
	return m
}

// Click handles clicks on the module output.
func (m *Module) click(e bar.Event) {
	current := m.count.Get().(int)
	switch e.Button {
	case bar.ButtonLeft, bar.ScrollDown, bar.ScrollLeft, bar.ButtonBack:
		current--
	case bar.ButtonRight, bar.ScrollUp, bar.ScrollRight, bar.ButtonForward:
		current++
	}
	m.count.Set(current)
}
