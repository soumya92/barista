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

/*
Package split provides a module that splits the output from an existing module
and sends it to two new modules. This can be useful for modules that display
a lot of information, by allowing some of the module output to be placed
elsewhere, typically somewhere in a group that only shows it when requested.

For example, an email module may support multiple folders, so it could be
configured to output one segment per folder, inbox coming first. It can then
be split at 1, to provide a summary for just inbox, and a detail for all other
folders, with the detail module placed in a collapsible group.

	labels := []string{"INBOX", "OUTBOX", "To-Do", "Follow-Up"}
	mail := mailProvider.New(labels...).
		Output(func(m mailProvider.Info) bar.Output {
			o := outputs.Group()
			for _, lbl := range labels {
				o.Append(outputs.Textf("%d", m[lbl]))
			}
			return o
		})
	inbox, others := split.SplitModule(mail, 1)
*/
package split // import "barista.run/modules/meta/split"

import (
	"sync"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/core"
)

type module struct {
	*value.Value
	start func() // called on Stream(), to ensure backing module is started
	index int    // index of split
	first bool   // whether this module shows segments before split
}

// Stream starts the module, and tries to stream the original module as well.
// Due to the sync.Once, the first of the two split modules to start will cause
// the original module to start as well, while the second will not affect it.
func (m *module) Stream(sink bar.Sink) {
	go m.start()
	for {
		next := m.Next()
		s, _ := m.Get().(bar.Segments)
		index := m.index
		if index > len(s) {
			index = len(s)
		}
		var out bar.Segments
		if m.first {
			out = s[:index]
		} else {
			out = s[index:]
		}
		sink(out)
		<-next
	}
}

// SplitModule splits the output from a module at index n, and returns two
// modules. The first module displays segments 0 through n (inclusive), while
// the second module displays all remaining segments. One or both of the modules
// will show an empty output if there are not enough segments.
func SplitModule(original bar.Module, n int) (first, rest bar.Module) {
	segments := new(value.Value)
	coreModule := core.NewModule(original)

	var once sync.Once
	start := func() {
		once.Do(func() {
			coreModule.Stream(func(s bar.Segments) { segments.Set(s) })
		})
	}

	first = &module{segments, start, n, true}
	rest = &module{segments, start, n, false}
	return first, rest
}
