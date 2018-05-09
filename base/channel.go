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

// Package base provides some helpers to make constructing bar modules easier.
package base

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
)

// Channel wraps a `chan bar.Output` with useful methods.
type Channel chan bar.Output

// NewChannel creates a new bar.Output channel.
func NewChannel() Channel {
	return Channel(make(chan bar.Output, 1))
}

// Clear hides the module from the bar.
func (c Channel) Clear() {
	c <- outputs.Empty()
}

// Output updates the module's output.
func (c Channel) Output(out bar.Output) {
	c <- out
}

// Error shows an urgent "Error" on the bar (or the full text if it fits)
// and closes the output channel, allowing a click to restart the module.
func (c Channel) Error(err error) bool {
	if err == nil {
		return false
	}
	c <- outputs.Error(err)
	close(c)
	return true
}
