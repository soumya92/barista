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
	multi.OutputTemplate(``{{.Prop2}}`),
)
*/
package multi

import (
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
)

// ModuleSet represents a multi-module consisting of many individual modules.
type ModuleSet struct {
	submodules []*base.Base
	updateFunc func()
	scheduler  *scheduler
	primaryIdx int
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
	module := base.New()
	key := len(m.submodules)
	module.OnUpdate(m.onDemandUpdate(key))
	m.submodules = append(m.submodules, module)
	return module
}

// Clear hides all modules from the bar.
func (m *ModuleSet) Clear() {
	for _, module := range m.submodules {
		module.Clear()
	}
}

// Error sets all modules to an error state.
func (m *ModuleSet) Error(err error) bool {
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
	if m.primaryIdx >= 0 {
		m.submodules[m.primaryIdx].Update()
		return
	}
	for _, module := range m.submodules {
		module.Update()
	}
}

// scheduler stores scheduled update information and applies
// it to the first submodule that becomes active. onDemandUpdate
// takes care of propagating the updates to other submodules.
type scheduler struct {
	when     time.Time
	delay    time.Duration
	interval time.Duration
}

func (s *scheduler) applyTo(b *base.Base) {
	switch {
	case !s.when.IsZero():
		b.UpdateAt(s.when)
	case s.delay > 0:
		b.UpdateAfter(s.delay)
	case s.interval > 0:
		b.UpdateEvery(s.interval)
	}
}

// UpdateAt schedules submodules for updating at a specific time.
func (m *ModuleSet) UpdateAt(when time.Time) {
	m.scheduler = &scheduler{when: when}
}

// UpdateAfter schedules submodules for updating after a delay.
func (m *ModuleSet) UpdateAfter(delay time.Duration) {
	m.scheduler = &scheduler{delay: delay}
}

// UpdateEvery schedules submodules for repeated updating at an interval.
func (m *ModuleSet) UpdateEvery(interval time.Duration) {
	m.scheduler = &scheduler{interval: interval}
}

// OnUpdate sets the function that will be run on each update.
// An initial update will occur when the first module begins streaming,
// and subsequent updates may be scheduled using timer.AfterFunc,
// a goroutine, or an asynchronous mechanism.
func (m *ModuleSet) OnUpdate(updateFunc func()) {
	m.updateFunc = updateFunc
}

// onDemandUpdate wraps the update func with deduplication logic to
// ensure that it's only called once between updates for all submodules.
func (m *ModuleSet) onDemandUpdate(key int) func() {
	return func() {
		// The first submodule to update is marked "primary".
		// Any calls to update actually end up calling update on the primary submodule,
		// and all update scheduling is performed on the primary submodule as well.
		if m.primaryIdx < 0 {
			m.primaryIdx = key
			if m.scheduler != nil {
				m.scheduler.applyTo(m.submodules[key])
				m.scheduler = nil
			}
		}
		if m.updateFunc != nil {
			m.updateFunc()
		}
	}
}
