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

// Package static provides a simple module that shows static content on the bar,
// with methods to set the content. In a pinch, this can be used to create
// buttons, or show additional information by setting the output from within
// a format function.
package static

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/value"
)

// Module represents a module that displays static content on the bar.
type Module struct {
	output value.Value
}

// Stream starts the module.
func (m *Module) Stream(sink bar.Sink) {
	for {
		next := m.output.Next()
		out, _ := m.output.Get().(bar.Output)
		sink.Output(out)
		<-next
	}
}

// Set sets the output to display.
func (m *Module) Set(out bar.Output) {
	m.output.Set(out)
}

// Clear sets an empty output.
func (m *Module) Clear() {
	m.output.Set(nil)
}

// New constructs a static module that displays the given output.
func New(initial bar.Output) *Module {
	m := new(Module)
	m.Set(initial)
	return m
}
