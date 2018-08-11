// Copyright 2018 Google Inc.
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

package core

import (
	"testing"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
	"github.com/stretchr/testify/require"
)

func nextOutput(t *testing.T, ch <-chan bar.Segments, formatAndArgs ...interface{}) bar.Segments {
	select {
	case out := <-ch:
		return out
	case <-time.After(time.Second):
		require.Fail(t, "No output on sink", formatAndArgs...)
	}
	return nil
}

func assertNoOutput(t *testing.T, ch <-chan bar.Segments, formatAndArgs ...interface{}) {
	select {
	case <-ch:
		require.Fail(t, "Unexpected output on sink", formatAndArgs...)
	case <-time.After(10 * time.Millisecond):
		// test passed.
	}
}

func chanSink() (<-chan bar.Segments, Sink) {
	ch := make(chan bar.Segments)
	return ch, func(s bar.Segments) { ch <- s }
}

func nullSink(bar.Segments) {}

func TestModule(t *testing.T) {
	tm := testModule.New(t)
	m := NewModule(tm)
	ch, sink := chanSink()

	tm.AssertNotStarted("before stream")
	go m.Stream(sink)
	tm.AssertStarted("after stream")
	assertNoOutput(t, ch, "on start")

	tm.Output(outputs.Text("test"))
	require.Equal(t, "test", nextOutput(t, ch)[0].Text())

	tm.Output(nil)
	require.Empty(t, nextOutput(t, ch))
}

func TestReplay(t *testing.T) {
	tm := testModule.New(t)
	m := NewModule(tm)
	ch, sink := chanSink()

	m.Replay()
	// If this didn't panic, test passed.

	go m.Stream(sink)
	tm.AssertStarted()
	assertNoOutput(t, ch, "on start with pending Replay")

	tm.Output(outputs.Text("foo"))
	require.Equal(t, "foo",
		nextOutput(t, ch, "on regular output")[0].Text())

	m.Replay()
	require.Equal(t, "foo",
		nextOutput(t, ch, "on replay")[0].Text())
}

type simpleModule struct{ returned chan bool }

func (s *simpleModule) Stream(sink bar.Sink) {
	sink(outputs.Text("foo"))
	s.returned <- true
}

type listenerModule struct{ returned, finished chan bool }

func (l *listenerModule) Stream(sink bar.Sink) {
	sink(outputs.Text("baz"))
	l.returned <- true
}

func (l *listenerModule) ModuleFinished() {
	l.finished <- true
}

func TestFinishListener(t *testing.T) {
	sm := &simpleModule{make(chan bool)}
	m := NewModule(sm)
	go m.Stream(nullSink)
	require.True(t, <-sm.returned, "After stream returns")

	lm := &listenerModule{make(chan bool), make(chan bool)}
	m = NewModule(lm)
	go m.Stream(nullSink)
	require.True(t, <-lm.returned, "After stream returns")
	require.True(t, <-lm.finished, "Finish listener is called")
}

type clickableModule struct {
	events           chan bar.Event
	finish, finished chan bool
}

func (c *clickableModule) Stream(sink bar.Sink) {
	<-c.finish
}

func (c *clickableModule) ModuleFinished() {
	c.finished <- true
}

func (c *clickableModule) Click(e bar.Event) {
	c.events <- e
}

func TestEvents(t *testing.T) {
	// Just a sanity check to make sure things don't panic.
	// None of the .Click() calls should cause a panic.
	// However, since the runLoop is in a different goroutine,
	// saying require.NotPanics(m.Click(...)) doesn't actually
	// assert what we want it to require.

	sm := &simpleModule{make(chan bool)}
	m := NewModule(sm)
	go m.Stream(nullSink)
	m.Click(bar.Event{Button: bar.ButtonLeft})
	<-sm.returned
	m.Click(bar.Event{Button: bar.ButtonLeft})

	lm := &listenerModule{make(chan bool), make(chan bool)}
	m = NewModule(lm)
	go m.Stream(nullSink)
	m.Click(bar.Event{Button: bar.ButtonLeft})
	<-lm.returned
	<-lm.finished
	m.Click(bar.Event{Button: bar.ButtonLeft})

	cm := &clickableModule{make(chan bar.Event), make(chan bool), make(chan bool)}
	m = NewModule(cm)
	go m.Stream(nullSink)
	m.Click(bar.Event{Button: bar.ButtonLeft})
	require.Equal(t, bar.ButtonLeft, (<-cm.events).Button)
	cm.finish <- true
	<-cm.finished
	m.Click(bar.Event{Button: bar.ButtonLeft})
	select {
	case <-cm.events:
		require.Fail(t, "module received event after finish")
	default:
	}
}

func TestRestart(t *testing.T) {
	tm := testModule.New(t)
	m := NewModule(tm)
	ch, sink := chanSink()

	tm.AssertNotStarted("before stream")
	go m.Stream(sink)
	tm.AssertStarted("after stream")
	assertNoOutput(t, ch, "on start")

	tm.Output(outputs.Group(
		outputs.Errorf("something went wrong"),
		outputs.Text("test"),
	))
	require.Error(t, nextOutput(t, ch)[0].GetError())

	tm.Close()
	assertNoOutput(t, ch, "On close")

	m.Click(bar.Event{Button: bar.ScrollUp})
	assertNoOutput(t, ch, "On scroll event")
	tm.AssertNotStarted("On scroll event")

	m.Click(bar.Event{Button: bar.ButtonMiddle})
	out := nextOutput(t, ch)
	require.Equal(t, 1, len(out), "Only non-error segments on restart")
	require.Equal(t, "test", out[0].Text())
	tm.AssertStarted("on middle click")
}
