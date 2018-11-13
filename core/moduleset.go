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

package core // import "barista.run/core"

import (
	"sync"

	"barista.run/bar"
	l "barista.run/logging"
	"barista.run/sink"
)

// ModuleSet is a group of modules. It provides a channel for identifying module
// updates, and methods to get the last output of the set or a specific module.
type ModuleSet struct {
	modules   []*Module
	updateCh  chan int
	outputs   []bar.Segments
	outputsMu sync.RWMutex
}

// NewModuleSet creates a ModuleSet with the given modules.
func NewModuleSet(modules []bar.Module) *ModuleSet {
	set := &ModuleSet{
		modules:  make([]*Module, len(modules)),
		outputs:  make([]bar.Segments, len(modules)),
		updateCh: make(chan int),
	}
	for i, m := range modules {
		l.Fine("%s added as %s[%d]", l.ID(m), l.ID(set), i)
		set.modules[i] = NewModule(m)
	}
	return set
}

// Stream starts streaming all modules and returns a channel that receives the
// index of the module any time one updates with new output.
func (m *ModuleSet) Stream() <-chan int {
	for i, mod := range m.modules {
		go mod.Stream(m.sinkFn(i))
	}
	return m.updateCh
}

func (m *ModuleSet) sinkFn(idx int) bar.Sink {
	return sink.Func(func(out bar.Segments) {
		l.Fine("%s new output from %s",
			l.ID(m), l.ID(m.modules[idx].original))
		m.outputsMu.Lock()
		m.outputs[idx] = out
		m.outputsMu.Unlock()
		m.updateCh <- idx
	})
}

// Len returns the number of modules in this ModuleSet.
func (m *ModuleSet) Len() int {
	return len(m.modules)
}

// LastOutput returns the last output from the module at a specific position.
// If the module has not yet updated, an empty output will be used.
func (m *ModuleSet) LastOutput(idx int) bar.Segments {
	m.outputsMu.RLock()
	defer m.outputsMu.RUnlock()
	return m.outputs[idx]
}

// LastOutputs returns the last output from all modules in order. The returned
// slice will have exactly Len() elements, and if a module has not yet updated
// an empty output will be placed in its position.
func (m *ModuleSet) LastOutputs() []bar.Segments {
	m.outputsMu.RLock()
	defer m.outputsMu.RUnlock()
	cp := make([]bar.Segments, len(m.outputs))
	copy(cp, m.outputs)
	return cp
}
