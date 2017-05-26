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

// Package base provides some helpers to make constructing bar modules easier.
package base

import (
	"os/exec"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
)

// Base is a simple module that satisfies the bar.Module interface, while adding
// some useful functions to make building modules on top somewhat simpler.
type Base struct {
	channel        chan bar.Output
	clickHandler   func(bar.Event)
	updateFunc     func()
	paused         bool
	updateOnResume bool
	outputOnResume bar.Output
	lastError      error
}

// Module implements bar's Module, Clickable, and Pausable,
// and adds a method to trigger updates on demand.
type Module interface {
	bar.Module
	bar.Clickable
	bar.Pausable
	Update()
}

// WithClickHandler extends Module to add support for a generic click handler
// that only receives the bar.Event. Most custom modules should extend Module
// with their own OnClick method that provides module-specific information.
type WithClickHandler interface {
	Module
	OnClick(func(bar.Event)) Module
}

// Stream starts up the worker goroutine, and channels its output to the bar.
func (b *Base) Stream() <-chan bar.Output {
	b.Resume()
	// Constructed when New is called, but is not directly exposed to extending
	// modules. Use Output or Clear to control the bar output.
	return b.channel
}

// Click handles click events from the bar.
// A middle click will always force an update, but if the module
// is currently in an error state, the configured click handler
// will be replaced by one that shows the error message using
// i3-nagbar on left click and updates the module on right click
func (b *Base) Click(e bar.Event) {
	if b.lastError == nil {
		if e.Button == bar.ButtonMiddle {
			b.Update()
		}
		if b.clickHandler != nil {
			b.clickHandler(e)
		}
		return
	}
	switch e.Button {
	case bar.ButtonRight:
	case bar.ButtonMiddle:
		b.lastError = nil
		b.Clear()
		b.Update()
	case bar.ButtonLeft:
		go exec.Command("i3-nagbar", "-m", b.lastError.Error()).Run()
	}
}

// Pause marks the module as paused, which suspends updates
// and outputs to the bar.
func (b *Base) Pause() {
	b.paused = true
}

// Resume continues normal updating of the module, and performs an
// immediate update if any updates occurred while the module was paused.
func (b *Base) Resume() {
	b.paused = false
	if b.outputOnResume != nil {
		b.Output(b.outputOnResume)
		b.outputOnResume = nil
	}
	if b.updateOnResume {
		b.updateOnResume = false
		b.Update()
	}
}

// Update marks the module as ready for an update.
// The actual update may not happen immediately, e.g. if the bar is hidden.
func (b *Base) Update() {
	if b.paused {
		b.updateOnResume = true
		return
	}
	go b.updateFunc()
}

// OnClick sets the click handler.
// This is a minimal default implementation; derived modules should implement an
// alternative OnClick method that exposes module-specific data to the handler function.
// Returns Module to allow bar.Add/bar.Run on the result.
func (b *Base) OnClick(handler func(bar.Event)) Module {
	b.clickHandler = handler
	return b
}

// New constructs a new base module.
func New() *Base {
	return &Base{
		channel:    make(chan bar.Output),
		updateFunc: func() {},
		// Modules start paused so that any modifications prior to Stream()
		// are not applied before the module has started.
		paused: true,
		// Trigger an initial update when Stream is first called.
		updateOnResume: true,
	}
}

// OnUpdate sets the function that will be called when the module needs
// to be update. That function can choose to call Output/Clear/Error to
// update the output, but is not required if no visual update is necessary.
// This method is only called while the bar is visible, to conserve resources
// when possible. For this reason, it is recommended that heavy update work,
// e.g. http requests, should happen here and not in an independent timer.
func (b *Base) OnUpdate(updateFunc func()) {
	b.updateFunc = updateFunc
}

// Clear hides the module from the bar.
func (b *Base) Clear() {
	b.Output(outputs.Empty())
}

// Output updates the module's output.
func (b *Base) Output(out bar.Output) {
	if b.paused {
		b.outputOnResume = out
		return
	}
	go func() { b.channel <- out }()
}

// Error shows an error on the bar.
// It shows an urgent "Error" on the bar (or the full text if it fits),
// and when clicked shows the full error using i3-nagbar.
func (b *Base) Error(err error) bool {
	if err == nil {
		return false
	}
	b.lastError = err
	b.Output(outputs.Error(err))
	return true
}

// Scheduler holds either a timer or a ticker that schedules
// the next update of the module. Since many modules need to
// schedule updates, adding support into base simplifies module
// code and works well with pause/resume.
type Scheduler struct {
	timer  *time.Timer
	ticker *time.Ticker
}

// Stop cancels the next scheduled update, used when the update
// frequency/schedule changes.
func (s Scheduler) Stop() {
	if s.timer != nil {
		s.timer.Stop()
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
}

// UpdateAt schedules the module for updating at a specific time.
func (b *Base) UpdateAt(when time.Time) Scheduler {
	return b.UpdateAfter(time.Until(when))
}

// UpdateAfter schedules the module for updating after a delay.
func (b *Base) UpdateAfter(delay time.Duration) Scheduler {
	return Scheduler{
		timer: time.AfterFunc(delay, b.Update),
	}
}

// UpdateEvery schedules the module for repeated updating at an interval.
func (b *Base) UpdateEvery(interval time.Duration) Scheduler {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			b.Update()
		}
	}()
	return Scheduler{ticker: ticker}
}
