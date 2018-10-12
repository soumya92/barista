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

// Package sink provides functions to create test sinks.
package sink // import "barista.run/base/sink"

import (
	"barista.run/bar"
	"barista.run/base/value"
)

// New creates a new sink and returns a channel that
// will emit any outputs sent to the sink.
func New() (<-chan bar.Output, bar.Sink) {
	return Buffered(0)
}

// Buffered creates a new buffered sink.
func Buffered(bufCount int) (<-chan bar.Output, bar.Sink) {
	ch := make(chan bar.Output, bufCount)
	return ch, func(o bar.Output) { ch <- o }
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
	go func(ch <-chan bar.Output, val *value.Value) {
		for o := range ch {
			val.Set(o)
		}
	}(ch, val)
	return val, sink
}
