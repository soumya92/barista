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

// Package slot provides multiple slots for a single module, allowing it to be
// moved between various positions on the bar. When used carefully, this can be
// useful for conveying limited information by re-ordering modules, but it has
// the potential to become too distracting if overused.
package slot

import (
	"sync"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/value"
	"github.com/soumya92/barista/sink"
)

// Slotter provides the ability to display the output of a module into named
// slots, and allows changing the active slot at runtime to effectively move
// the module on the bar.
type Slotter struct {
	module     bar.Module
	stream     sync.Once
	activeSlot value.Value // of string

	sink       bar.Sink
	lastOutput *value.Value // of bar.Segments
}

// New creates a slotter for the given module. The module is 'consumed' by
// this operation and should not be used except through slots created from the
// returned Slotter.
func New(m bar.Module) *Slotter {
	s := &Slotter{module: m}
	s.lastOutput, s.sink = sink.Value()
	s.activeSlot.Set("")
	return s
}

// Slot creates a named slot for the module output.
func (s *Slotter) Slot(name string) bar.Module {
	return &slotModule{s, name}
}

// Activate moves the module output to the named slot.
func (s *Slotter) Activate(slotName string) {
	s.activeSlot.Set(slotName)
}

type slotModule struct {
	*Slotter
	slotName string
}

func (s *slotModule) Stream(sink bar.Sink) {
	go s.stream.Do(func() { s.module.Stream(s.sink) })

	activeSub, done := s.activeSlot.Subscribe()
	defer done()
	active := s.activeSlot.Get().(string)

	outputSub, done := s.lastOutput.Subscribe()
	defer done()
	out := s.lastOutput.Get().(bar.Segments)

	hasOutput := false
	outputChanged := true

	for {
		if active == s.slotName {
			if !hasOutput || outputChanged {
				sink(out)
				hasOutput = true
			}
		} else {
			if hasOutput {
				sink(nil)
				hasOutput = false
			}
		}
		outputChanged = false
		select {
		case <-activeSub:
			active = s.activeSlot.Get().(string)
		case <-outputSub:
			out = s.lastOutput.Get().(bar.Segments)
			outputChanged = true
		}
	}
}
