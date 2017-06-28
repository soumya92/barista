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
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

// TestError tests that base.Error(error) behaves as expected: a nil error
// returns false and does nothing, a non-nil error returns true and updates
// the module's output.
func TestError(t *testing.T) {
	b := New()
	o := testModule.NewOutputTester(t, b)

	assert.True(t, b.Error(fmt.Errorf("test error")), "returns true for non-nil error")
	err := o.AssertError("on error")
	assert.Equal(t, "test error", err, "error message is displayed on the output")

	assert.False(t, b.Error(nil), "returns false for nil error")
	o.AssertNoOutput("on nil error")
}

// TestUpdateAndScheduler tests that update functions (including nil)
// are correctly handled, and that the returned scheduler works
// as intended.
func TestUpdateAndScheduler(t *testing.T) {
	scheduler.TestMode(true)
	b := New()
	o := testModule.NewOutputTester(t, b)

	assert.NotPanics(t, b.Update, "Calling update without setting OnUpdate")
	b.OnUpdate(nil)
	assert.NotPanics(t, b.Update, "Calling update with nil OnUpdate")

	updateCalled := false
	b.OnUpdate(func() {
		updateCalled = true
		b.Output(bar.Output{bar.NewSegment("test")})
	})

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

	assertNoUpdate("on setting OnUpdate")
	b.Update()
	assertUpdate("on calling Update")

	b.Schedule().Every(time.Minute)
	assertNoUpdate("when scheduling")
	scheduler.NextTick()
	assertUpdate("On next tick")
	scheduler.NextTick()
	assertUpdate("On next tick")
	b.Schedule().Stop()
	scheduler.NextTick()
	assertNoUpdate("when stopped")
}

// TestPauseResume tests that pause/resume work as expected, i.e. no
// updates occur while the module is paused, and calls to update are
// queued up properly and execute on resume.
func TestPauseResume(t *testing.T) {
	b := New()
	updateCalled := false
	b.OnUpdate(func() {
		updateCalled = true
		b.Output(outputs.Text("test"))
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
	oldOut := outputs.Text("output")
	b.Output(oldOut)
	o.AssertNoOutput("while paused")

	b.Clear()
	o.AssertNoOutput("while paused")

	newOut := outputs.Text("new")
	b.Output(newOut)
	o.AssertNoOutput("while paused")

	b.Resume()
	out := o.AssertOutput("from calling output while paused")
	assert.Equal(t, newOut, out, "updates with last output")
	o.AssertNoOutput("only last output emitted on resume")
}

// TestClickUpdates tests the update behaviour on click events,
// for both the normal case and the error case.
func TestClickUpdates(t *testing.T) {
	b := New()

	updateCalled := false
	b.OnUpdate(func() {
		updateCalled = true
		b.Output(outputs.Text("test"))
	})
	var lastClickEvent *bar.Event
	b.OnClick(func(e bar.Event) {
		lastClickEvent = &e
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

	assertClicked := func(button bar.Button, message string) {
		assert.Equal(t, button, lastClickEvent.Button, message)
		lastClickEvent = nil
	}

	assertNotClicked := func(message string) {
		assert.Nil(t, lastClickEvent, message)
	}

	clickEvent := func(button bar.Button) bar.Event {
		return bar.Event{Button: button}
	}

	assertUpdate("when started")

	b.Click(clickEvent(bar.ButtonMiddle))
	assertUpdate("on middle click")
	assertClicked(bar.ButtonMiddle, "click event passed through")

	for _, btn := range []bar.Button{
		bar.ButtonLeft, bar.ButtonRight, bar.ButtonBack, bar.ButtonForward,
		bar.ScrollUp, bar.ScrollDown, bar.ScrollLeft, bar.ScrollRight,
	} {
		b.Click(clickEvent(btn))
		assertNoUpdate("no special handling for other buttons")
		assertClicked(btn, "click event passed through")
	}

	b.Error(fmt.Errorf("test error"))
	o.AssertOutput("on error")
	assertNotClicked("when no click event")

	b.Click(clickEvent(bar.ButtonRight))
	out := o.AssertOutput("on right click")
	assert.Empty(t, out, "clears on right click when error'd")
	assertUpdate("after clearing error")
	assertNotClicked("when clearing error")

	b.Error(fmt.Errorf("test error"))
	o.AssertOutput("on error")

	b.Click(clickEvent(bar.ButtonMiddle))
	out = o.AssertOutput("on middle click")
	assert.Empty(t, out, "clears on middle click when error'd")
	assertUpdate("after clearing error")
	assertNotClicked("when clearing error")

	b.Error(fmt.Errorf("test error"))
	o.AssertOutput("on error")

	for _, btn := range []bar.Button{
		bar.ButtonBack, bar.ButtonForward,
		bar.ScrollUp, bar.ScrollDown, bar.ScrollLeft, bar.ScrollRight,
	} {
		b.Click(clickEvent(btn))
		assertNoUpdate("no special handling for other buttons")
		assertNotClicked("when in error state")
	}

	// TODO: Test shelling out to i3-nagbar on left-click.
}

func hammerOnBase(b *Base, done chan<- interface{}) {
	for i := 0; i < 10; i++ {
		b.OnUpdate(func() {
			b.Output(outputs.Text("update"))
		})
		b.Output(outputs.Text("test"))
		b.Update()
		b.OnClick(func(e bar.Event) {})
		b.Clear()
		b.Error(fmt.Errorf("test error"))
	}
	done <- nil
}

func devNull(b *Base) {
	for range b.Stream() {
	}
}

// Simple tests to ensure that base is locked appropriately.
// This test is primarily meant to run under the race detector.
func TestLocking(t *testing.T) {
	b := New()
	// Prevent output channel from filling up.
	go devNull(b)
	doneChan := make(chan interface{})
	// Detect any data races.
	for i := 0; i < 5; i++ {
		go hammerOnBase(b, doneChan)
	}
	for i := 0; i < 5; i++ {
		<-doneChan
	}
	// Track updates
	updateCalled := false
	b.OnUpdate(func() {
		updateCalled = true
		doneChan <- nil
	})
	// Ensure locking works correctly.
	b.Lock()
	go func() {
		b.Lock()
		doneChan <- nil
	}()
	select {
	case <-doneChan:
		assert.Fail(t, "lock did not wait for unlock!")
	default:
	}
	b.Unlock()
	select {
	case <-doneChan:
	case <-time.After(10 * time.Millisecond):
		assert.Fail(t, "lock did not return after unlock!")
	}
	assert.False(t, updateCalled, "on simple unlocking")

	b.UnlockAndUpdate()
	<-doneChan
	assert.True(t, updateCalled, "UnlockAndUpdate")

	testFatalUnlockError(t, "unlock")
	testFatalUnlockError(t, "unlockAndUpdate")
}

func testFatalUnlockError(t *testing.T, testName string) {
	out, err := exec.Command(os.Args[0], "FatalUnlockError", testName).CombinedOutput()
	if err == nil || !strings.Contains(string(out), "unlocked") {
		t.Errorf("%s: did not find failure with message about unlocked lock: %s\n%s\n", testName, err, out)
	}
}

func init() {
	if len(os.Args) == 3 && os.Args[1] == "FatalUnlockError" {
		switch os.Args[2] {
		case "unlock":
			b := New()
			b.Unlock()
		case "unlockAndUpdate":
			b := New()
			b.UnlockAndUpdate()
		}
		os.Exit(0)
	}
}
