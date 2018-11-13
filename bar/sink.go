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

package bar

// Output updates the module's output on the bar.
func (s Sink) Output(o Output) {
	s(o)
}

// Error is a convenience method that returns false and does nothing
// when given a nil error, and outputs an error segment and returns
// true when a non-nil error is given.
func (s Sink) Error(e error) bool {
	if e != nil {
		s(ErrorSegment(e))
		return true
	}
	return false
}
