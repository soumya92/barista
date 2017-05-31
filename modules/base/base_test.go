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
	testModule "github.com/soumya92/barista/testing/module"
)

// TestError tests that base.Error(error) behaves as expected: a nil error
// returns false and does nothing, a non-nil error returns true and updates
// the module's output.
func TestError(t *testing.T) {
	b := New()
	o := testModule.NewOutputTester(t, b)

	assert.True(t, b.Error(fmt.Errorf("test error")), "returns true for non-nil error")
	out := o.AssertOutput("on error")
	assert.Equal(t, "test error", out[0]["full_text"], "error message is displayed on the output")

	assert.False(t, b.Error(nil), "returns false for nil error")
	o.AssertNoOutput("on nil error")
}

// TestPauseResume tests that pause/resume work as expected, i.e. no
// updates occur while the module is paused, and calls to update are
// queued up properly and execute on resume.
func TestPauseResume(t *testing.T) {
	b := New()
	updateCalled := false
	b.OnUpdate(func() {
		updateCalled = true
		b.Output(bar.Output{bar.NewSegment("test")})
	})
	o := testModule.NewOutputTester(t, b)

	assertUpdate := func(message string) bar.Output {
		out := o.AssertOutput(message)
		assert.True(t, updateCalled, message)
		updateCalled = false
		return out
	}

	assertNoUpdate := func(message string) {
		assert.False(t, updateCalled, message)
		o.AssertNoOutput(message)
	}

	assertUpdate("when started")
	assertNoUpdate("only once when started")

	b.Update()
	assertUpdate("while resumed")

	b.Pause()
	b.Update()
	assertNoUpdate("paused")

	b.Update()
	assertNoUpdate("repeatedly calling update while paused")

	b.Resume()
	assertUpdate("resolved on resume")
	assertNoUpdate("coalesced to single update on resume")

	b.Pause()
	b.Resume()
	assertNoUpdate("on resume if update not called while paused")

	b.Pause()
	oldOut := bar.Output{bar.NewSegment("output")}
	b.Output(oldOut)
	o.AssertNoOutput("while paused")

	b.Clear()
	o.AssertNoOutput("while paused")

	newOut := bar.Output{bar.NewSegment("new")}
	b.Output(newOut)
	o.AssertNoOutput("while paused")

	b.Resume()
	out := o.AssertOutput("from calling output while paused")
	assert.Equal(t, newOut, out, "updates with last output")
	o.AssertNoOutput("only last output emitted on resume")
}

// TestSchedulers tests that scheduling updates for a base module work as expected.
// It tests that schedulers returned from Update* can be cancelled, and that no updates
// occur while the module is paused.
func TestSchedulers(t *testing.T) {
	b := New()
	updateCalled := false
	b.OnUpdate(func() {
		updateCalled = true
		b.Output(bar.Output{bar.NewSegment("test")})
	})
	o := testModule.NewOutputTester(t, b)

	assertUpdate := func(message string) {
		o.AssertOutput(message)
		assert.True(t, updateCalled, message)
		updateCalled = false
	}

	assertNoUpdate := func(message string) {
		o.AssertNoOutput(message)
		assert.False(t, updateCalled, message)
	}

	assertUpdate("when started")

	b.UpdateAfter(5 * time.Millisecond).Stop()
	assertNoUpdate("if scheduler was cancelled")

	b.UpdateEvery(5 * time.Millisecond).Stop()
	assertNoUpdate("if scheduler was cancelled")

	b.UpdateAt(time.Now().Add(5 * time.Millisecond)).Stop()
	assertNoUpdate("if scheduler was cancelled")

	b.UpdateAfter(10 * time.Millisecond)
	assertUpdate("after interval elapses")

	sch := b.UpdateAfter(5 * time.Millisecond)
	b.Pause()
	assertNoUpdate("if paused before interval elapses")

	b.Resume()
	assertUpdate("on resume")

	sch.Stop()
	assertNoUpdate("when elapsed scheduler is stopped")

	b.UpdateAt(time.Now().Add(15 * time.Millisecond))
	b.Pause()
	b.Resume()
	assertUpdate("pause + resume within interval is no-op")

	sch = b.UpdateEvery(5 * time.Millisecond)
	assertUpdate("while resumed")
	b.Pause()
	assertNoUpdate("while paused")
	assertNoUpdate("no repeats while paused")
	b.Resume()
	assertUpdate("when resumed")
	assertUpdate("repeat after resume")
	assertUpdate("repeating while resumed")
	sch.Stop()
	assertNoUpdate("after scheduler is stopped")
}
