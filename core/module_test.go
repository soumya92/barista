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

	"barista.run/bar"
	"barista.run/outputs"
	"barista.run/sink"
	testModule "barista.run/testing/module"

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

func TestModule(t *testing.T) {
	tm := testModule.New(t)
	tm.SkipClickHandlers() // needed for nil output.
	m := NewModule(tm)
	ch, sink := sink.New()

	tm.AssertNotStarted("before stream")
	go m.Stream(sink)
	tm.AssertStarted("after stream")
	assertNoOutput(t, ch, "on start")

	tm.Output(outputs.Text("test"))
	txt, _ := nextOutput(t, ch)[0].Content()
	require.Equal(t, "test", txt)

	tm.Output(nil)
	require.Empty(t, nextOutput(t, ch))
}

func TestReplay(t *testing.T) {
	tm := testModule.New(t)
	m := NewModule(tm)
	ch, sink := sink.New()

	m.Replay()
	// If this didn't panic, test passed.

	go m.Stream(sink)
	tm.AssertStarted()
	assertNoOutput(t, ch, "on start with pending Replay")

	tm.Output(outputs.Text("foo"))
	txt, _ := nextOutput(t, ch, "on regular output")[0].Content()
	require.Equal(t, "foo", txt)

	m.Replay()
	txt, _ = nextOutput(t, ch, "on replay")[0].Content()
	require.Equal(t, "foo", txt)
}

type simpleModule struct{ returned chan bool }

func (s *simpleModule) Stream(sink bar.Sink) {
	sink.Output(outputs.Text("foo"))
	s.returned <- true
}

func TestRestart(t *testing.T) {
	tm := testModule.New(t)
	m := NewModule(tm)
	ch, sink := sink.New()

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
	out := nextOutput(t, ch, "On close (to set click handlers)")

	out[0].Click(bar.Event{Button: bar.ScrollUp})
	assertNoOutput(t, ch, "On scroll event")
	tm.AssertNotStarted("On scroll event")
	tm.AssertNotClicked("After close")

	out[0].Click(bar.Event{Button: bar.ButtonMiddle})
	tm.AssertNotClicked("When restarted")
	out = nextOutput(t, ch)
	require.Equal(t, 1, len(out), "Only non-error segments on restart")
	txt, _ := out[0].Content()
	require.Equal(t, "test", txt)
	tm.AssertStarted("on middle click")
}
