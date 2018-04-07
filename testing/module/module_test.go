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

package module

import (
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
)

func finishedWithin(f func(), timeout time.Duration) bool {
	doneChan := make(chan interface{})
	go func() {
		f()
		doneChan <- nil
	}()
	select {
	case <-doneChan:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestSimple(t *testing.T) {
	m := New(t)
	m.AssertNotStarted("Initially not started")
	initialOutput := outputs.Text("hello")
	assert.True(t,
		finishedWithin(func() { m.Output(initialOutput) }, time.Second),
		"Output does not block before start")
	ch := m.Stream()
	m.AssertStarted("Started when streaming starts")
	assert.Panics(t, func() { m.Stream() }, "Panics when streamed again")

	secondOutput := outputs.Text("world")
	m.Output(secondOutput)
	var out bar.Output
	assert.True(t,
		finishedWithin(func() { out = <-ch }, time.Second),
		"Read from channel does not block when output is buffered")
	assert.Equal(t, initialOutput, out, "Outputs are returned sequentially")
	out = <-ch
	assert.Equal(t, secondOutput, out, "Outputs are not dropped")
	assert.False(t,
		finishedWithin(func() { <-ch }, 10*time.Millisecond),
		"Read from channel blocks when no output")
}

func TestOutputBuffer(t *testing.T) {
	m := New(t)
	out1 := outputs.Text("1")
	out2 := outputs.Text("2")
	out3 := outputs.Text("3")
	ch := m.Stream()
	m.Output(out1)
	m.Output(out2)
	actual1 := <-ch
	actual2 := <-ch
	assert.Equal(t, out1, actual1, "buffered write")
	assert.Equal(t, out2, actual2, "buffered write")
	m.Output(out3)
	actual3 := <-ch
	assert.Equal(t, out3, actual3, "buffered write")
}

func TestClick(t *testing.T) {
	positiveTimeout = 10 * time.Millisecond

	m := New(t)
	evt1 := bar.Event{X: 2}
	evt2 := bar.Event{Y: 2}
	evt3 := bar.Event{X: 1, Y: 1}

	m.AssertNotClicked("no events initially")
	m.Click(evt1)
	m.Click(evt2)
	evt := m.AssertClicked("when module is clicked")
	assert.Equal(t, evt1, evt, "events are ordered")
	evt = m.AssertClicked("when module is clicked")
	assert.Equal(t, evt2, evt, "events are buffered")
	m.AssertNotClicked("events cleared after assertions")
	m.Click(evt3)
	evt = m.AssertClicked("events resume after being cleared")
	assert.Equal(t, evt3, evt, "new events received")
	m.AssertNotClicked("no extra events")

	fakeT := &testing.T{}
	m = New(fakeT)
	m.AssertClicked("fails when not clicked")
	assert.True(t, fakeT.Failed(), "AssertClicked when not clicked")

	fakeT = &testing.T{}
	m = New(fakeT)
	m.Click(evt1)
	m.AssertNotClicked("fails when clicked")
	assert.True(t, fakeT.Failed(), "AssertNotClicked when clicked")
}

func TestPause(t *testing.T) {
	positiveTimeout = 10 * time.Millisecond

	m := New(t)
	m.Pause()
	m.AssertPaused("paused")
	m.AssertNoPauseResume("invocation consumed on assertion")
	m.Pause()
	m.Pause()
	m.AssertPaused("repeated pause")
	m.AssertPaused("repeated pause")
	m.AssertNoPauseResume("repeated invocations consumed")
	m.Resume()
	m.AssertResumed("resumed")
	m.Resume()
	m.Resume()
	m.AssertResumed("repeated resume")
	m.AssertResumed("repeated resume")
	m.AssertNoPauseResume("repeated invocations consumed")
	m.Resume()
	m.Resume()
	m.Pause()
	m.Resume()
	m.Resume()
	m.Pause()
	m.AssertResumed("ordering")
	m.AssertResumed("ordering")
	m.AssertPaused("ordering")
	m.AssertResumed("ordering")
	m.AssertResumed("ordering")
	m.AssertPaused("ordering")
	m.AssertNoPauseResume("consumed")

	fakeT := &testing.T{}
	m = New(fakeT)
	m.AssertPaused("fails when not paused")
	assert.True(t, fakeT.Failed(), "AssertPaused when not paused")

	fakeT = &testing.T{}
	m = New(fakeT)
	m.Resume()
	m.AssertPaused("fails when not paused")
	assert.True(t, fakeT.Failed(), "AssertPaused when not paused")

	fakeT = &testing.T{}
	m = New(fakeT)
	m.AssertResumed("fails when not resumed")
	assert.True(t, fakeT.Failed(), "AssertResumed when not resumed")

	fakeT = &testing.T{}
	m = New(fakeT)
	m.Pause()
	m.AssertResumed("fails when not resumed")
	assert.True(t, fakeT.Failed(), "AssertResumed when not resumed")

	fakeT = &testing.T{}
	m = New(fakeT)
	m.Pause()
	m.AssertNoPauseResume("fails when paused")
	assert.True(t, fakeT.Failed(), "AssertNoPauseResume when paused")

	fakeT = &testing.T{}
	m = New(fakeT)
	m.Resume()
	m.AssertNoPauseResume("fails when resumed")
	assert.True(t, fakeT.Failed(), "AssertNoPauseResume when resumed")
}

func TestReset(t *testing.T) {
	m := New(t)
	m.Pause()
	m.Resume()
	m.Pause()
	m.Output(outputs.Empty())
	m.Output(outputs.Text("test"))
	m.Click(bar.Event{})
	m.Stream()
	m.AssertStarted("some assertions before reset")
	m.AssertPaused("some assertions before reset")
	m.Reset()
	m.AssertNotClicked("reset resets events")
	m.AssertNoPauseResume("reset resets pause/resume")
	var ch <-chan bar.Output
	assert.NotPanics(t, func() { ch = m.Stream() }, "start after reset")
	assert.False(t,
		finishedWithin(func() { <-ch }, 10*time.Millisecond),
		"No previous output sent over channel")
}

func TestOutputTester(t *testing.T) {
	positiveTimeout = 10 * time.Millisecond

	m := New(t)
	o := NewOutputTester(t, m)
	m.AssertStarted("by output tester")
	o.AssertNoOutput("no output")
	testOut := outputs.Text("test")
	m.Output(testOut)
	actualOut := o.AssertOutput("has output")
	assert.Equal(t, testOut.Segments(), actualOut,
		"output passed through")
	m.Output(outputs.Empty())
	o.AssertEmpty("on empty output")

	m.Output(outputs.Errorf("error"))
	errStr := o.AssertError("on error output")
	assert.Equal(t, "error", errStr, "error string passed through")

	m.Output(outputs.Text("1"))
	m.Output(outputs.Text("2"))
	m.Output(outputs.Text("3"))
	o.Drain()
	testOut = outputs.Text("4")
	m.Output(testOut)
	actualOut = o.AssertOutput("has output")
	assert.Equal(t, testOut.Segments(), actualOut,
		"drain removes previous outputs")

	fakeT := &testing.T{}
	m = New(fakeT)
	o = NewOutputTester(fakeT, m)
	assert.False(t, fakeT.Failed(), "before failing assertion")
	o.AssertOutput("no output")
	assert.True(t, fakeT.Failed(), "AssertOutput without output")

	fakeT = &testing.T{}
	m = New(fakeT)
	o = NewOutputTester(fakeT, m)
	m.Output(testOut)
	assert.False(t, fakeT.Failed(), "before failing assertion")
	o.AssertNoOutput("with output")
	assert.True(t, fakeT.Failed(), "AssertNoOutput with output")

	fakeT = &testing.T{}
	m = New(fakeT)
	o = NewOutputTester(fakeT, m)
	m.Output(testOut)
	assert.False(t, fakeT.Failed(), "before failing assertion")
	o.AssertEmpty("with non-empty output")
	assert.True(t, fakeT.Failed(), "AssertEmpty with non-empty output")

	fakeT = &testing.T{}
	m = New(fakeT)
	o = NewOutputTester(fakeT, m)
	m.Output(testOut)
	assert.False(t, fakeT.Failed(), "before failing assertion")
	o.AssertError("with non-error output")
	assert.True(t, fakeT.Failed(), "AssertError with non-error output")

	fakeT = &testing.T{}
	m = New(fakeT)
	o = NewOutputTester(fakeT, m)
	m.Output(outputs.Empty())
	assert.False(t, fakeT.Failed(), "before failing assertion")
	o.AssertError("with empty output")
	assert.True(t, fakeT.Failed(), "AssertError with empty output")

	fakeT = &testing.T{}
	m = New(fakeT)
	o = NewOutputTester(fakeT, m)
	assert.False(t, fakeT.Failed(), "before failing assertion")
	o.AssertError("with no output")
	assert.True(t, fakeT.Failed(), "AssertError with empty output")
}
