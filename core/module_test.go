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
	"barista.run/timing"

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

func TestTimedOutput(t *testing.T) {
	timing.TestMode()
	tm := testModule.New(t).SkipClickHandlers()
	m := NewModule(tm)
	ch, sink := sink.New()

	tm.AssertNotStarted("before stream")
	go m.Stream(sink)
	tm.AssertStarted("after stream")
	assertNoOutput(t, ch, "on start")

	start := timing.Now()
	tm.Output(outputs.Repeat(func(now time.Time) bar.Output {
		return outputs.Textf("%v", now.Sub(start))
	}).Every(time.Minute))
	txt, _ := nextOutput(t, ch)[0].Content()
	require.Equal(t, "0s", txt)

	assertNoOutput(t, ch, "until time advances")
	timing.AdvanceBy(30 * time.Second)
	assertNoOutput(t, ch, "until refresh time elapses")
	timing.AdvanceBy(45 * time.Second)

	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "1m0s", txt)

	timing.NextTick()
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "2m0s", txt)

	tm.Output(bar.TextSegment("foo"))
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "foo", txt)

	timing.NextTick()
	assertNoOutput(t, ch, "When no longer using timed output")
}

func TestTimedOutputRealTime(t *testing.T) {
	timing.ExitTestMode()

	tm := testModule.New(t).SkipClickHandlers()
	m := NewModule(tm)
	ch, sink := sink.New()

	tm.AssertNotStarted("before stream")
	go m.Stream(sink)
	tm.AssertStarted("after stream")
	assertNoOutput(t, ch, "on start")

	start := timing.Now()
	tm.Output(outputs.Repeat(func(now time.Time) bar.Output {
		ms := timing.Now().Sub(start).Seconds() * 1000.0
		return outputs.Textf("%.0f", ms/100)
	}).Every(100 * time.Millisecond))

	txt, _ := nextOutput(t, ch)[0].Content()
	require.Equal(t, "0", txt)

	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "1", txt)

	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "2", txt)

	tm.Output(bar.TextSegment("foo"))
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "foo", txt)

	assertNoOutput(t, ch, "When no longer using timed output")

	// Test rapid updates for data races.
	tm.Output(outputs.Repeat(func(now time.Time) bar.Output {
		return outputs.Text(now.Format("15:04:05.000000"))
	}).Every(time.Nanosecond))

	nextOutput(t, ch)
	nextOutput(t, ch)
	nextOutput(t, ch)
	nextOutput(t, ch)

	tm.Output(bar.TextSegment("foo"))
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "foo", txt)

	assertNoOutput(t, ch, "When no longer using timed output")
}
