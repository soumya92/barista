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
	"github.com/soumya92/barista/sink"
)

func finishedWithin(f func(), timeout time.Duration) bool {
	doneChan := make(chan struct{})
	go func() {
		f()
		doneChan <- struct{}{}
	}()
	select {
	case <-doneChan:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestSimple(t *testing.T) {
	positiveTimeout = time.Second
	m := New(t)
	m.AssertNotStarted("Initially not started")
	initialOutput := outputs.Text("hello")
	assert.Panics(t, func() { m.Output(initialOutput) },
		"Panics when output without stream")
	assert.Panics(t, func() { m.Click(bar.Event{}) },
		"Panics when clicked without stream")
	ch, s := sink.New()
	go m.Stream(s)
	m.AssertStarted("Started when streaming starts")
	assert.Panics(t, func() { m.Stream(sink.Null()) }, "Panics when streamed again")
	assert.True(t,
		finishedWithin(func() { m.Output(initialOutput) }, time.Second),
		"Output does not block")

	secondOutput := outputs.Text("world")
	m.OutputText("world")
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
	positiveTimeout = time.Second
	m := New(t)
	out1 := outputs.Text("1")
	out2 := outputs.Text("2")
	out3 := outputs.Text("3")
	ch, s := sink.New()
	go m.Stream(s)
	m.AssertStarted()
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
	positiveTimeout = time.Second

	m := New(t)
	evt1 := bar.Event{X: 2}
	evt2 := bar.Event{Y: 2}
	evt3 := bar.Event{X: 1, Y: 1}
	go m.Stream(sink.Null())
	m.AssertStarted()

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

	positiveTimeout = 10 * time.Millisecond
	fakeT := &testing.T{}
	m = New(fakeT)
	m.AssertClicked("fails when not started")
	assert.True(t, fakeT.Failed(), "AssertClicked when not started")

	fakeT = &testing.T{}
	m = New(fakeT)
	go m.Stream(sink.Null())
	m.AssertStarted()
	m.AssertClicked("fails when not clicked")
	assert.True(t, fakeT.Failed(), "AssertClicked when not clicked")

	fakeT = &testing.T{}
	m = New(fakeT)
	go m.Stream(sink.Null())
	m.AssertStarted()
	m.Click(evt1)
	m.AssertNotClicked("fails when clicked")
	assert.True(t, fakeT.Failed(), "AssertNotClicked when clicked")
}

func TestClose(t *testing.T) {
	m := New(t)
	go m.Stream(sink.Null())
	m.AssertStarted()

	assert.Panics(t, func() { m.Stream(sink.Null()) },
		"already streaming")
	m.Close()
	assert.Panics(t, func() { m.OutputText("foo") },
		"output after close")
	assert.NotPanics(t, func() { go m.Stream(sink.Null()) },
		"after closing module")
	m.AssertStarted()
	assert.NotPanics(t, func() { m.OutputText("foo") },
		"output after restarting")
}

func TestStarted(t *testing.T) {
	positiveTimeout = time.Second

	m := New(t)
	go m.Stream(sink.Null())
	assert.True(t, finishedWithin(func() { m.AssertStarted() }, time.Second),
		"AssertStarted when module starts streaming")
	assert.True(t, finishedWithin(func() { m.AssertStarted() }, time.Second),
		"AssertStarted when module is already streaming")

	signalChan := make(chan bool)
	doneChan := make(chan bool)
	m2 := New(t)
	go func() {
		signalChan <- true
		m2.AssertStarted()
		doneChan <- true
	}()
	<-signalChan
	go m2.Stream(sink.Null())
	assert.True(t,
		finishedWithin(func() { <-doneChan }, time.Second),
		"AssertStarted after streaming")

	positiveTimeout = 10 * time.Millisecond
	fakeT := &testing.T{}
	m3 := New(fakeT)
	assert.False(t, fakeT.Failed())
	m3.AssertStarted()
	assert.True(t, fakeT.Failed(),
		"AssertStarted fails if module is not streamed")
}
