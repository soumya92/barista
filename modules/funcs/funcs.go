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

// Package funcs provides the ability to construct i3bar modules from simple Funcs.
package funcs // import "barista.run/modules/funcs"

import (
	"barista.run/base/notifier"
	"barista.run/timing"
	"time"

	"barista.run/bar"
)

// Func receives a bar.Sink and uses it for output.
type Func func(bar.Sink)

// Once constructs a bar module that runs the given function once.
// Useful if the function loops internally.
func Once(f Func) *OnceModule {
	return &OnceModule{Func: f}
}

// OnceModule represents a bar.Module that runs a function once.
// If the function sets an error output, it will be restarted on
// the next click.
type OnceModule struct {
	Func
}

// Stream starts the module.
func (o *OnceModule) Stream(s bar.Sink) {
	forever := make(chan struct{})
	o.Func(s)
	<-forever
}

// OnClick constructs a bar module that runs the given function
// when clicked. The function is given a Channel to allow
// multiple outputs (e.g. Loading... Done), and when the function
// returns, the next click will call it again.
func OnClick(f Func) *OnclickModule {
	return &OnclickModule{f}
}

// OnclickModule represents a bar.Module that runs a function and
// marks the module as finished, causing the next click to start the
// module again.
type OnclickModule struct {
	Func
}

// Stream starts the module.
func (o OnclickModule) Stream(s bar.Sink) {
	o.Func(s)
}

// Every constructs a bar module that repeatedly runs the given function.
// Useful if the function needs to poll a resource for output.
func Every(d time.Duration, f Func) *RepeatingModule {
	r := &RepeatingModule{fn: f, d: d}
	r.scheduler = timing.NewScheduler().Every(d)
	r.notifyFn, r.notifyCh = notifier.New()

	return r
}

// RepeatingModule represents a bar.Module that runs a function at a fixed
// interval (while accounting for bar paused/resumed state).
type RepeatingModule struct {
	notifyCh  <-chan struct{}
	notifyFn  func()
	fn        Func
	d         time.Duration
	scheduler *timing.Scheduler
}

func (r *RepeatingModule) GetScheduler() *timing.Scheduler {
	return r.scheduler
}

// Stream starts the module.
func (r *RepeatingModule) Stream(s bar.Sink) {
	r.fn(s)
	for {
		select {
		case <-r.notifyCh:
			r.scheduler.Close()
			r.scheduler = timing.NewScheduler().Every(r.d)
			r.fn(s)
		case <-r.scheduler.C:
			r.fn(s)
		}
	}
}

// Refresh forces a refresh of the data being displayed.
func (r *RepeatingModule) Refresh() {
	r.notifyFn()
}
