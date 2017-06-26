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
)

// Module is an interface that allows functions to display their module output.
type Module interface {
	Output(bar.Output)
	Clear()
	Error(error) bool
}

// Func uses the Module interface for output.
type Func func(Module)

// Once constructs a bar module that runs the given function once.
// Useful if the function loops internally.
func Once(f Func) base.WithClickHandler {
	b := base.New()
	f(b)
	return b
}

// Every constructs a bar module that repeatedly runs the given function.
// Useful if the function needs to poll a resource for output.
func Every(d time.Duration, f Func) base.WithClickHandler {
	b := base.New()
	b.OnUpdate(func() { f(b) })
	b.Schedule().Every(d)
	return b
}
