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

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"
)

func TestSchedulers(t *testing.T) {
	ch := make(chan interface{})
	doFunc := func() {
		ch <- nil
	}

	assertCalled := func(message string) {
		select {
		case <-ch:
		case <-time.After(time.Second):
			assert.Fail(t, "doFunc was not called", message)
		}
	}

	assertNotCalled := func(message string) {
		select {
		case <-ch:
			assert.Fail(t, "doFunc was called", message)
		case <-time.After(10 * time.Millisecond):
		}
	}

	sch := Do(doFunc)
	assertNotCalled("when not scheduled")

	sch.After(5 * time.Millisecond).Stop()
	assertNotCalled("when stopped")

	sch.Every(5 * time.Millisecond).Stop()
	assertNotCalled("when stopped")

	sch.At(Now().Add(5 * time.Millisecond)).Stop()
	assertNotCalled("when stopped")

	sch.After(10 * time.Millisecond)
	assertCalled("after interval elapses")

	sch.Stop()
	assertNotCalled("when elapsed scheduler is stopped")

	sch.Stop()
	assertNotCalled("when elapsed scheduler is stopped again")

	sch = Do(doFunc).Every(5 * time.Millisecond)
	assertCalled("after interval elapses")
	assertCalled("after interval elapses")
	assertCalled("after interval elapses")

	sch.Stop()
	assertNotCalled("when stopped")

	sch.After(5 * time.Millisecond)
	assertCalled("after delay elapses")
	assertNotCalled("after first trigger")

}
