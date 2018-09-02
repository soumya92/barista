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

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/sink"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/testing/fail"
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
	m := New(t).SkipClickHandlers()
	m.AssertNotStarted("Initially not started")
	initialOutput := outputs.Text("hello")
	require.Panics(t, func() { m.Output(initialOutput) },
		"Panics when output without stream")
	ch, s := sink.New()
	go m.Stream(s)
	m.AssertStarted("Started when streaming starts")
	require.Panics(t, func() { m.Stream(sink.Null()) }, "Panics when streamed again")
	require.True(t,
		finishedWithin(func() { m.Output(initialOutput) }, time.Second),
		"Output does not block")

	secondOutput := outputs.Text("world")
	m.OutputText("world")
	var out bar.Output
	require.True(t,
		finishedWithin(func() { out = <-ch }, time.Second),
		"Read from channel does not block when output is buffered")
	require.Equal(t, initialOutput.Segments(), out.Segments(),
		"Outputs are returned sequentially")
	out = <-ch
	require.Equal(t, secondOutput.Segments(), out.Segments(),
		"Outputs are not dropped")
	require.False(t,
		finishedWithin(func() { <-ch }, 10*time.Millisecond),
		"Read from channel blocks when no output")
}

func TestOutputBuffer(t *testing.T) {
	positiveTimeout = time.Second
	m := New(t).SkipClickHandlers()
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
	require.Equal(t, out1.Segments(), actual1.Segments(), "buffered write")
	require.Equal(t, out2.Segments(), actual2.Segments(), "buffered write")
	m.Output(out3)
	actual3 := <-ch
	require.Equal(t, out3.Segments(), actual3.Segments(), "buffered write")
}

func TestClick(t *testing.T) {
	positiveTimeout = time.Second
	ch, s := sink.New()

	m := New(t)
	evt1 := bar.Event{X: 2}
	evt2 := bar.Event{Y: 2}
	evt3 := bar.Event{X: 1, Y: 1}
	go m.Stream(s)
	m.AssertStarted()
	m.OutputText("foo")

	m.AssertNotClicked("no events initially")
	nextOut := (<-ch).Segments()[0]
	nextOut.Click(evt1)
	nextOut.Click(evt2)
	evt := m.AssertClicked("when module is clicked")
	require.Equal(t, evt1, evt, "events are ordered")
	evt = m.AssertClicked("when module is clicked")
	require.Equal(t, evt2, evt, "events are buffered")
	m.AssertNotClicked("events cleared after assertions")
	nextOut.Click(evt3)
	evt = m.AssertClicked("events resume after being cleared")
	require.Equal(t, evt3, evt, "new events received")
	m.AssertNotClicked("no extra events")

	positiveTimeout = 10 * time.Millisecond
	fail.AssertFails(t, func(fakeT *testing.T) {
		m = New(fakeT)
		m.AssertClicked("fails when not started")
	}, "AssertClicked when not started")

	fail.AssertFails(t, func(fakeT *testing.T) {
		m = New(fakeT)
		go m.Stream(sink.Null())
		m.AssertStarted()
		m.AssertClicked("fails when not clicked")
	}, "AssertClicked when not clicked")

	fail.AssertFails(t, func(fakeT *testing.T) {
		m = New(fakeT)
		ch, s := sink.New()
		go m.Stream(s)
		m.AssertStarted()
		m.OutputText("foo")
		(<-ch).Segments()[0].Click(evt1)
		m.AssertNotClicked("fails when clicked")
	}, "AssertNotClicked when clicked")
}

func TestClose(t *testing.T) {
	m := New(t)
	go m.Stream(sink.Null())
	m.AssertStarted()

	require.Panics(t, func() { m.Stream(sink.Null()) },
		"already streaming")
	m.Close()
	require.Panics(t, func() { m.OutputText("foo") },
		"output after close")
	require.NotPanics(t, func() { go m.Stream(sink.Null()) },
		"after closing module")
	m.AssertStarted()
	require.NotPanics(t, func() { m.OutputText("foo") },
		"output after restarting")
}

func TestStarted(t *testing.T) {
	positiveTimeout = time.Second

	m := New(t)
	go m.Stream(sink.Null())
	require.True(t, finishedWithin(func() { m.AssertStarted() }, time.Second),
		"AssertStarted when module starts streaming")
	require.True(t, finishedWithin(func() { m.AssertStarted() }, time.Second),
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
	require.True(t,
		finishedWithin(func() { <-doneChan }, time.Second),
		"AssertStarted after streaming")

	positiveTimeout = 10 * time.Millisecond
	fail.AssertFails(t, func(fakeT *testing.T) {
		m3 := New(fakeT)
		require.False(t, fakeT.Failed())
		m3.AssertStarted()
	}, "AssertStarted fails if module is not streamed")
}
