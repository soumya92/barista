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

// Package sink provides functions to create sinks.
package sink // import "barista.run/sink"

import (
	"barista.run/bar"
	"barista.run/base/value"
)

// Func creates a bar.Sink that sends sends output in the form of Segments.
func Func(s func(bar.Segments)) bar.Sink {
	return func(o bar.Output) {
		if o == nil {
			s(nil)
			return
		}
		s(o.Segments())
	}
}

// New creates a new sink and returns a channel that
// will emit any outputs sent to the sink.
func New() (<-chan bar.Segments, bar.Sink) {
	return Buffered(0)
}

// Buffered creates a new buffered sink.
func Buffered(bufCount int) (<-chan bar.Segments, bar.Sink) {
	ch := make(chan bar.Segments, bufCount)
	return ch, Func(func(o bar.Segments) { ch <- o })
}

// Null returns a sink that swallows any output sent to it.
func Null() bar.Sink {
	ch, sink := New()
	go func() {
		for range ch {
		}
	}()
	return sink
}

// Value returns a sink that sends output to a base.Value.
func Value() (*value.Value, bar.Sink) {
	ch, sink := New()
	val := new(value.Value)
	go func(ch <-chan bar.Segments, val *value.Value) {
		for o := range ch {
			val.Set(o)
		}
	}(ch, val)
	val.Set(bar.Segments(nil))
	return val, sink
}
