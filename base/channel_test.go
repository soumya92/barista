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

package base

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/testing/output"
)

func TestChannelOutput(t *testing.T) {
	ch := NewChannel()
	ch.Output(bar.TextSegment("test"))
	out := <-ch
	output.New(t, out).AssertText([]string{"test"},
		"Output is passed through")
}

func TestChannelClear(t *testing.T) {
	ch := NewChannel()
	ch.Clear()
	out := <-ch
	output.New(t, out).AssertEmpty("Clear sends empty output")
}

func TestChannelError(t *testing.T) {
	ch := NewChannel()
	assert.False(t, ch.Error(nil))
	select {
	case <-ch:
		assert.Fail(t, "Expected no output on nil error")
	case <-time.After(10 * time.Millisecond):
		// Test passed.
	}

	assert.True(t, ch.Error(fmt.Errorf("foobar")))
	out := <-ch
	errStrs := output.New(t, out).AssertError("on error")
	assert.Equal(t, []string{"foobar"}, errStrs,
		"Error string passed through")
	assert.Panics(t, func() { ch.Output(bar.TextSegment("test")) },
		"Cannot send to channel after error")
	assert.Panics(t, func() { ch.Clear() },
		"Cannot clear channel after error")
}
