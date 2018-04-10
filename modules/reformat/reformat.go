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
Package reformat provides a module that "wraps" an existing module and transforms it's output.
This can be useful for adding extra formatting simple bar modules.

For example, a time module might use strftime-style format strings,
which don't allow for colours or borders. You can add those using reformat:

 t := localtime.New(...)
 r := reformat.New(t, func(o bar.Output) bar.Output {
   return o.Background("red").Padding(20)
 })
*/
package reformat

import (
	"github.com/soumya92/barista/bar"
)

// FormatFunc takes the module's output and returns a modified version.
type FormatFunc func(bar.Output) bar.Output

// module stores the original module and the re-formatting function.
type module struct {
	bar.Module
	Formatter FormatFunc
}

// New wraps an existing bar.Module and applies formatFunc to it's output.
func New(original bar.Module, formatFunc FormatFunc) bar.Module {
	return &module{original, formatFunc}
}

// Stream sets up the formatting pipeline and returns a channel for the bar.
func (m *module) Stream() <-chan bar.Output {
	reformatted := make(chan bar.Output)
	go format(m.Module.Stream(), m.Formatter, reformatted)
	return reformatted
}

// Click passes through the click event if supported by the wrapped module.
func (m *module) Click(e bar.Event) {
	if clickable, ok := m.Module.(bar.Clickable); ok {
		clickable.Click(e)
	}
}

// Pause passes through the pause event if supported by the wrapped module.
func (m *module) Pause() {
	if pausable, ok := m.Module.(bar.Pausable); ok {
		pausable.Pause()
	}
}

// Resume passes through the resume event if supported by the wrapped module.
func (m *module) Resume() {
	if pausable, ok := m.Module.(bar.Pausable); ok {
		pausable.Resume()
	}
}

// format takes input from a channel, formats it using the format function,
// and outputs it to a different channel.
func format(input <-chan bar.Output, f FormatFunc, output chan<- bar.Output) {
	for out := range input {
		output <- f(out)
	}
}
