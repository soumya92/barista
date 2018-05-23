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
package counter

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// Module represents a "counter" module that displays a count
// in the given format, and adjusts the count on click/scroll.
// This module exemplifies the event-based architecture of barista.
type Module struct {
	count  base.Value // of int
	format base.Value // of string
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
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

// Click handles clicks on the module output.
func (m *Module) Click(e bar.Event) {
	current := m.count.Get().(int)
	switch e.Button {
	case bar.ButtonLeft, bar.ScrollDown, bar.ScrollLeft, bar.ButtonBack:
		current--
	case bar.ButtonRight, bar.ScrollUp, bar.ScrollRight, bar.ButtonForward:
		current++
	}
	m.count.Set(current)
}

func (m *Module) worker(ch base.Channel) {
	count := m.count.Get().(int)
	sCount := m.count.Subscribe()

	format := m.format.Get().(string)
	sFormat := m.format.Subscribe()

	for {
		ch.Output(outputs.Textf(format, count))
		select {
		case <-sCount:
			count = m.count.Get().(int)
		case <-sFormat:
			format = m.format.Get().(string)
		}
	}
}
