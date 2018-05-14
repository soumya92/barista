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
package funcs

import (
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/timing"
)

// Channel provides methods for functions to send output to the bar.
type Channel interface {
	// Output updates the module's output.
	Output(bar.Output)
	// Clear hides the module from the bar.
	Clear()
	// Error shows an error and restarts the module on click.
	Error(error) bool
}

type channel struct {
	base.Channel
	finished bool
}

func (c *channel) Error(err error) bool {
	c.finished = c.Channel.Error(err)
	return c.finished
}

// Func receives a Channel and uses it for output.
type Func func(Channel)

// Once constructs a bar module that runs the given function once.
// Useful if the function loops internally.
func Once(f Func) *OnceModule {
	return &OnceModule{Func: f}
}

// OnceModule represents a bar.Module that runs a function once.
// If the function sets an error output, it will be restarted on
// the next click.
type OnceModule struct {
	base.SimpleClickHandler
	Func
}

// Stream starts the module.
func (o *OnceModule) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go o.Func(ch)
	return ch
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
func (o OnclickModule) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go func() {
		wrappedCh := &channel{Channel: ch}
		o.Func(wrappedCh)
		if !wrappedCh.finished {
			close(ch)
		}
	}()
	return ch
}

// Every constructs a bar module that repeatedly runs the given function.
// Useful if the function needs to poll a resource for output.
func Every(d time.Duration, f Func) *RepeatingModule {
	return &RepeatingModule{fn: f, duration: d}
}

// RepeatingModule represents a bar.Module that runs a function at a fixed
// interval (while accounting for bar paused/resumed state).
type RepeatingModule struct {
	base.SimpleClickHandler
	fn       Func
	duration time.Duration
}

// Stream starts the module.
func (r *RepeatingModule) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	wrappedCh := &channel{Channel: ch}
	sch := timing.NewScheduler().Every(r.duration)
	go func() {
		for {
			r.fn(wrappedCh)
			if wrappedCh.finished {
				// The next click will call stream again, so return
				// from this goroutine so that the other instance can
				// run normally.
				sch.Stop()
				return
			}
			<-sch.Tick()
		}
	}()
	return ch
}
