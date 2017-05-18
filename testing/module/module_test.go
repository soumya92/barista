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
	assert.False(t, m.started, "Initially not started")
	initialOutput := bar.Output{bar.NewSegment("hello")}
	assert.True(t,
		finishedWithin(func() { m.Output(initialOutput) }, time.Second),
		"Output does not block before start")
	ch := m.Stream()
	m.AssertStarted("Started when streaming starts")
	assert.Panics(t, func() { m.Stream() }, "Panics when streamed again")

	secondOutput := bar.Output{bar.NewSegment("world")}
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
	out1 := bar.Output{bar.NewSegment("1")}
	out2 := bar.Output{bar.NewSegment("2")}
	out3 := bar.Output{bar.NewSegment("3")}
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
	m := New(t)
	evt1 := bar.Event{X: 2}
	evt2 := bar.Event{Y: 2}
	evt3 := bar.Event{X: 1, Y: 1}

	m.AssertNotClicked("no events initially")
	m.Click(evt1)
	m.Click(evt2)
	m.AssertClicked(evt1, "events are ordered")
	m.AssertClicked(evt2, "events are buffered")
	m.AssertNotClicked("events cleared after assertions")
	m.Click(evt3)
	m.AssertClicked(evt3, "events resumed after cleared")
	m.AssertNotClicked("no extra events")
}

func TestPause(t *testing.T) {
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
}

func TestReset(t *testing.T) {
	m := New(t)
	m.Pause()
	m.Resume()
	m.Pause()
	m.Output(bar.Output{})
	m.Output(bar.Output{bar.NewSegment("test")})
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
