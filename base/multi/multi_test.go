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

package multi

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	testModule "github.com/soumya92/barista/testing/module"
)

func TestModuleSetOutputs(t *testing.T) {
	m := NewModuleSet()

	tester := testModule.NewOutputTester(t, m.New())
	tester.AssertNoOutput("without ModuleSet interaction")

	m.Clear()
	tester.AssertEmpty("when ModuleSet is cleared")

	err := fmt.Errorf("test error")
	assert.True(t, m.Error(err), "error returns true for non-nil error")
	actualOut := tester.AssertOutput("on ModuleSet error")
	assert.Equal(t, "test error", actualOut[0].Text(),
		"error message sent to submodule")

	assert.False(t, m.Error(nil), "error returns false for nil error")
	tester.AssertNoOutput("nil error should not update submodules")
}

var updateChan = make(chan interface{})

func updateFunc() { updateChan <- nil }
func assertNotUpdated(t *testing.T, message string) {
	select {
	case <-updateChan:
		assert.Fail(t, "expected no update", message)
	case <-time.After(10 * time.Millisecond):
	}
}
func assertUpdated(t *testing.T, message string) {
	select {
	case <-updateChan:
	case <-time.After(time.Second):
		assert.Fail(t, "expected an update", message)
	}
}

func TestUpdates(t *testing.T) {
	m := NewModuleSet()
	m.OnUpdate(updateFunc)

	m.Update()
	assertNotUpdated(t, "when no submodules exist")

	sub1 := m.New()
	m.Update()
	assertNotUpdated(t, "when submodule hasn't started")

	sub1.Stream()
	assertUpdated(t, "when submodule is first started")

	m.Update()
	assertUpdated(t, "with started submodule")

	sub2 := m.New()
	assertNotUpdated(t, "when creating a new submodule")

	m.Update()
	assertUpdated(t, "with additional submodule")

	sub2.Stream()
	assertUpdated(t, "on new submodule stream")

	sub1.Pause()
	sub2.Pause()
	m.Update()
	assertNotUpdated(t, "when submodules are paused")

	sub1.Resume()
	sub2.Resume()
	assertUpdated(t, "when submodules are resumed")

	sub1.Pause()
	sub2.Pause()

	sub1.Resume()
	sub2.Resume()
	assertNotUpdated(t, "when resumed if not updated while paused")
}

func TestMultipleSubmodules(t *testing.T) {
	m := NewModuleSet()
	m.OnUpdate(updateFunc)

	m.Update()
	assertNotUpdated(t, "when no submodules exist")

	sub1 := m.New()
	sub2 := m.New()
	sub3 := m.New()

	m.Update()
	assertNotUpdated(t, "when submodule hasn't started")

	sub1.Stream()
	sub2.Stream()
	sub3.Stream()
	for i := 0; i < 3; i++ {
		assertUpdated(t, "when submodule is first started")
	}
	assertNotUpdated(t, "when no submodule is updated")

	sub2.Update()
	assertUpdated(t, "when a submodule is updated")
	assertNotUpdated(t, "when no submodule is updated")
}
