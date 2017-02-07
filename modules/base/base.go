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

	"github.com/google/barista/bar"
	"github.com/google/barista/bar/outputs"
)

// Base is a simple module that satisfies the bar.Module interface, while adding
// some useful functions to make building modules on top somewhat simpler.
type Base struct {
	channel      chan *bar.Output
	clickHandler func(bar.Event)
	worker       func() error
	lastError    error
}

// Module adds the ability to set a click handler on a bar.Module. Extending
// modules should return a base.Module rather than a base.Base.
type Module interface {
	Stream() <-chan *bar.Output
	Click(e bar.Event)
	OnClick(func(bar.Event))
}

// Stream starts up the worker goroutine, and channels its output to the bar.
func (b *Base) Stream() <-chan *bar.Output {
	b.startWorker()
	// Constructed when New is called, but is not directly exposed to extending
	// modules. Use Output or Clear to control the bar output.
	return b.channel
}

// startWorker starts the worker goroutine (if any).
func (b *Base) startWorker() {
	if b.worker == nil {
		return
	}
	go func(b *Base) {
		if err := b.worker(); err != nil {
			b.lastError = err
			b.Error(err)
		}
	}(b)
}

// Click handles click events from the bar,
// and restarts the module's worker on a middle click.
func (b *Base) Click(e bar.Event) {
	if b.lastError != nil {
		if e.Button == bar.ButtonMiddle || e.Button == bar.ButtonRight {
			b.lastError = nil
			b.startWorker()
		}
		if e.Button == bar.ButtonLeft {
			// TODO: Use dbus.
			go exec.Command("notify-send", b.lastError.Error()).Run()
		}
		return
	}
	if b.clickHandler != nil {
		b.clickHandler(e)
	}
}

// OnClick sets the click handler.
// This is a minimal default implementation; derived modules should implement an
// alternative OnClick method that exposes module-specific data to the handler function.
func (b *Base) OnClick(handler func(bar.Event)) {
	b.clickHandler = handler
}

// New constructs a new base module.
func New() *Base {
	return &Base{
		channel: make(chan *bar.Output),
	}
}

// Clear hides the module from the bar.
func (b *Base) Clear() {
	go func() { b.channel <- nil }()
}

// Output updates the module's output.
func (b *Base) Output(out *bar.Output) {
	go func() { b.channel <- out }()
}

// SetWorker sets a worker function that will be run in a goroutine.
// Useful if the module requires continuous background work.
func (b *Base) SetWorker(worker func() error) {
	b.worker = worker
}

// Error shows an error on the bar.
// This method currently replaces the module output, but may later be modified to
// use i3-nagbar or similar.
// It also marks the module as error'd so that a middle click restarts the worker.
func (b *Base) Error(err error) {
	b.Output(outputs.Error(err))
}
