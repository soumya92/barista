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

package funcs

import (
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/base/scheduler"
)

var funcChan chan interface{}

func signal(m Module) {
	funcChan <- nil
}

func signalled() bool {
	select {
	case <-funcChan:
		return true
	case <-time.After(10 * time.Millisecond):
	}
	return false
}

func TestOneShot(t *testing.T) {
	assert := assert.New(t)
	funcChan = make(chan interface{}, 10)

	module := Once(signal)
	assert.False(signalled(), "Function isn't called until module starts streaming")

	module.Stream()
	assert.True(signalled(), "Function called when streaming")

	assert.False(signalled(), "Function is never called again")
	assert.False(signalled(), "Function is never called again")
	assert.False(signalled(), "Function is never called again")
}

func TestRepeated(t *testing.T) {
	assert := assert.New(t)
	scheduler.TestMode(true)
	funcChan = make(chan interface{}, 10)

	module := Every(time.Minute, signal)
	assert.False(signalled(), "Function isn't called until module starts streaming")

	module.Stream()
	assert.True(signalled(), "Function called when streaming")
	assert.False(signalled(), "Function is not called again until next tick")

	scheduler.NextTick()
	assert.True(signalled(), "Function is called on next tick")
}
