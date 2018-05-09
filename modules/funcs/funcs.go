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

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
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
func Once(f Func) base.SimpleClickHandlerModule {
	return &once{Func: f}
}

type once struct {
	base.SimpleClickHandler
	Func
}

func (o *once) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go o.Func(ch)
	return ch
}

// Every constructs a bar module that repeatedly runs the given function.
// Useful if the function needs to poll a resource for output.
func Every(d time.Duration, f Func) base.SimpleClickHandlerModule {
	return &every{f: f, d: d}
}

type every struct {
	base.SimpleClickHandler
	f Func
	d time.Duration
}

func (e *every) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	wrappedCh := &channel{Channel: ch}
	sch := barista.Schedule().Every(e.d)
	go func() {
		for {
			e.f(wrappedCh)
			if wrappedCh.finished {
				// The next click will call stream again, so return
				// from this goroutine so that the other instance can
				// run normally.
				sch.Stop()
				return
			}
			sch.Wait()
		}
	}()
	return ch
}
