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

On the module side, it's mostly the same as base.Module,
with an additional "key" on each method.

e.g. Output(key, output), Clear(key), OnClick(key, func), SetWorker(func).

On the user side, a multi module provides access to it's "submodules",
so the code would look something like:

m := SomeMultiModule(key1, key2)
bar.Run(m.Get(key1), m.Get(key2))

Because multi-modules introduce awkward syntax in the bar,
they should only be used if there is a very clear advantage.

For example, splitting the time and date is not worthy of a multi module
since running a simple timer is cheap.

Splitting email or system information is, because those are expensive,
and should be bundled as much as possible.
*/
package multi

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
)

// ModuleSet represents a multi-module consisting of many individual modules.
type ModuleSet struct {
	modules      map[interface{}]*base.Base
	clickHandler func(interface{}, bar.Event)
	worker       func() error
	running      bool
}

// Module is the public interface for ModuleSet.
// It only allows getting individual modules, nothing else.
// It should be provided to the user for use in their bar.
type Module interface {
	Get(key interface{}) base.Module
	AddAll(*bar.I3Bar)
}

// OnClick sets the click handler.
func (m *ModuleSet) OnClick(handler func(interface{}, bar.Event)) {
	m.clickHandler = handler
	for key, module := range m.modules {
		if handler == nil {
			module.OnClick(nil)
			continue
		}
		module.OnClick(func(e bar.Event) {
			handler(key, e)
		})
	}
}

// NewModuleSet constructs a new multi-module.
func NewModuleSet() *ModuleSet {
	return &ModuleSet{
		modules: make(map[interface{}]*base.Base),
	}
}

// Add adds new keys to the module set.
func (m *ModuleSet) Add(keys ...interface{}) {
	for _, k := range keys {
		module := base.New()
		if m.clickHandler != nil {
			module.OnClick(func(e bar.Event) {
				m.clickHandler(k, e)
			})
		}
		if m.worker != nil {
			module.SetWorker(m.onDemandWorker)
		}
		m.modules[k] = module
	}
}

// Clear hides a module from the bar.
func (m *ModuleSet) Clear(key interface{}) {
	if module, ok := m.modules[key]; ok {
		module.Clear()
	}
}

// ClearAll hides all modules from the bar.
func (m *ModuleSet) ClearAll() {
	for _, module := range m.modules {
		module.Clear()
	}
}

// Output updates a single module's output.
func (m *ModuleSet) Output(key interface{}, out *bar.Output) {
	if module, ok := m.modules[key]; ok {
		module.Output(out)
	}
}

// SetWorker sets a worker function that will be run in a goroutine.
// Useful if the module requires continuous background work.
// The worker will be started whenever the first module begins streaming.
func (m *ModuleSet) SetWorker(worker func() error) {
	m.worker = worker
	for _, module := range m.modules {
		module.SetWorker(m.onDemandWorker)
	}
}

// Get gets a single module from the list of modules.
func (m *ModuleSet) Get(key interface{}) base.Module {
	return m.modules[key]
}

// AddAll adds all submodules (in indeterminate order) to the given bar.
func (m *ModuleSet) AddAll(bar *bar.I3Bar) {
	for _, module := range m.modules {
		bar.Add(module)
	}
}

func (m *ModuleSet) onDemandWorker() error {
	// Only one copy of the worker will run, others will be no-ops.
	if m.running {
		return nil
	}
	m.running = true
	// This will block as long as the worker is running.
	err := m.worker()
	m.running = false
	// Only one module will display the error. Pretend that's WAI ;)
	return err
}
