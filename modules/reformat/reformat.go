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

// Stream starts the wrapped module and formats the output before sending
// it to the original bar.Sink.
func (m *module) Stream(s bar.Sink) {
	m.Module.Stream(reformatSink(s, m.Formatter))
}

// Click passes through the click event if supported by the wrapped module.
func (m *module) Click(e bar.Event) {
	if clickable, ok := m.Module.(bar.Clickable); ok {
		clickable.Click(e)
	}
}

func reformatSink(orig bar.Sink, formatter FormatFunc) bar.Sink {
	return func(o bar.Output) {
		orig.Output(formatter(o))
	}
}
