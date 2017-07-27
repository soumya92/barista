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
Package multi provides the ability for a single source to update multiple outputs.

For example, the gmail API supports getting the unread count for multiple labels,
but users might prefer a separate output for each label.

Multi module supports an update function that will only be run once per update
across all submodules.

It's up to extending modules to decide how to expose the submodule creation,
but one simple way to do so is to have OutputFunc/OutputTemplate return new
submodules with that output template. This would allow for code like:

 multi := multimodule.New(... config ...)
 bar.Run(
   multi.OutputTemplate(`{{.Prop1}}`),
   othermodule,
   multi.OutputTemplate(`{{.Prop2}}`),
 )
*/
package multi

import (
	"sync"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
)

// ModuleSet represents a multi-module consisting of many individual modules.
type ModuleSet struct {
	submodules []*base.Base
	updateFunc func()
	primaryIdx int
	mutex      sync.Mutex
}

// NewModuleSet constructs a new multi-module.
func NewModuleSet() *ModuleSet {
	// A negative primaryIdx means no submodule has updated yet.
	return &ModuleSet{primaryIdx: -1}
}

// Submodule exposes the base module interface with a click handler,
// and some additional function to control the output.
type Submodule interface {
	base.WithClickHandler
	Clear()
	Output(out bar.Output)
	Error(err error) bool
}

// New creates a submodule
func (m *ModuleSet) New() Submodule {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	module := base.New()
	key := len(m.submodules)
	module.OnUpdate(m.onDemandUpdate(key))
	m.submodules = append(m.submodules, module)
	return module
}

// Clear hides all modules from the bar.
func (m *ModuleSet) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, module := range m.submodules {
		module.Clear()
	}
}

// Error sets all modules to an error state.
func (m *ModuleSet) Error(err error) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if err == nil {
		return false
	}
	for _, module := range m.submodules {
		module.Error(err)
	}
	return true
}

// Update marks submodules as ready for an update.
func (m *ModuleSet) Update() {
	m.mutex.Lock()
	primaryIdx := m.primaryIdx
	m.mutex.Unlock()
	if primaryIdx >= 0 {
		m.submodules[primaryIdx].Update()
		return
	}
	m.mutex.Lock()
	submodules := m.submodules
	m.mutex.Unlock()
	for _, module := range submodules {
		module.Update()
	}
}

// OnUpdate sets the function that will be run on each update.
// An initial update will occur when the first module begins streaming,
// and subsequent updates may be scheduled using timer.AfterFunc,
// a goroutine, or an asynchronous mechanism.
func (m *ModuleSet) OnUpdate(updateFunc func()) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.updateFunc = updateFunc
}

// onDemandUpdate wraps the update func with deduplication logic to
// ensure that it's only called once between updates for all submodules.
func (m *ModuleSet) onDemandUpdate(key int) func() {
	return func() {
		m.mutex.Lock()
		// The first submodule to update is marked "primary".
		// Any calls to update actually end up calling update on the primary submodule,
		// and all update scheduling is performed on the primary submodule as well.
		if m.primaryIdx < 0 {
			m.primaryIdx = key
		}
		updateFunc := m.updateFunc
		m.mutex.Unlock()
		if updateFunc != nil {
			updateFunc()
		}
	}
}
