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
	"errors"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	"barista.run/sink"
	testModule "barista.run/testing/module"
	"barista.run/testing/notifier"
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

	require.NotPanics(t, func() { m.Replay() })

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

	m.Replay()
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "2m0s", txt)

	timing.NextTick()
	txt, _ = nextOutput(t, ch, "TimedOutput after Replay")[0].Content()
	require.Equal(t, "3m0s", txt)

	tm.Output(outputs.Group(bar.TextSegment("foo")))
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "foo", txt)

	timing.NextTick()
	assertNoOutput(t, ch, "When no longer using timed output")

	start = timing.Now()
	tm.Output(outputs.Repeat(func(now time.Time) bar.Output {
		return outputs.Textf("%v", now.Sub(start))
	}).Every(time.Minute))

	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "0s", txt)

	timing.NextTick()
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "1m0s", txt)

	tm.Close()
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "1m0s", txt)

	timing.NextTick()
	assertNoOutput(t, ch, "TimedOutput stops on close")

	timing.AdvanceBy(time.Hour)

	m.Replay()
	txt, _ = nextOutput(t, ch)[0].Content()
	require.Equal(t, "1m0s", txt)

	timing.NextTick()
	assertNoOutput(t, ch, "TimedOutput remains stopped after replay")
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

type refreshableModule struct {
	*testModule.TestModule
	refreshCh chan<- struct{}
}

func (r refreshableModule) Refresh() {
	r.refreshCh <- struct{}{}
}

func TestRefresh(t *testing.T) {
	timing.TestMode()
	refreshCh := make(chan struct{}, 1)
	tm := refreshableModule{testModule.New(t), refreshCh}
	m := NewModule(tm)
	ch, sink := sink.New()
	go m.Stream(sink)
	tm.AssertStarted()

	tm.Output(outputs.Text("foo"))
	out := nextOutput(t, ch, "on regular output")

	out[0].Click(bar.Event{Button: bar.ButtonLeft})
	notifier.AssertNoUpdate(t, refreshCh, "On left-click")
	tm.AssertClicked("on left-click")

	out[0].Click(bar.Event{Button: bar.ButtonMiddle})
	notifier.AssertNotified(t, refreshCh, "On middle-click")
	tm.AssertNotClicked("on middle-click")

	m.Replay()
	out = nextOutput(t, ch, "on replay")

	out[0].Click(bar.Event{Button: bar.ButtonLeft})
	notifier.AssertNoUpdate(t, refreshCh, "On left-click after replay")
	tm.AssertClicked("on left-click")

	out[0].Click(bar.Event{Button: bar.ButtonMiddle})
	notifier.AssertNotified(t, refreshCh, "On middle-click after replay")
	tm.AssertNotClicked("on middle-click")

	tm.Output(outputs.Repeat(func(now time.Time) bar.Output {
		return outputs.Textf(now.In(time.UTC).Format("15:04"))
	}).Every(time.Minute))

	out = nextOutput(t, ch, "on timed output")
	txt, _ := out[0].Content()
	require.Equal(t, "20:47", txt)
	out[0].Click(bar.Event{Button: bar.ButtonMiddle})
	notifier.AssertNotified(t, refreshCh, "middle-click of timed output")

	timing.NextTick()
	out = nextOutput(t, ch, "on timed output tick")
	txt, _ = out[0].Content()
	require.Equal(t, "20:48", txt)
	out[0].Click(bar.Event{Button: bar.ButtonMiddle})
	notifier.AssertNotified(t, refreshCh, "middle-click after tick")

	tm.Output(bar.Segments{
		bar.ErrorSegment(errors.New("foo")),
		bar.TextSegment("baz"),
	})

	out = nextOutput(t, ch, "on error output")
	out[0].Click(bar.Event{Button: bar.ButtonLeft})
	notifier.AssertNotified(t, refreshCh, "left-click on running module error")
	tm.AssertNotClicked("left-click on error segment")

	out[1].Click(bar.Event{Button: bar.ButtonLeft})
	notifier.AssertNoUpdate(t, refreshCh, "left-click on non-error segment")
	tm.AssertClicked("left-click handled normally")

	tm.Close()
	out = nextOutput(t, ch, "switch handlers on finish")
	out[1].Click(bar.Event{Button: bar.ButtonMiddle})
	notifier.AssertNoUpdate(t, refreshCh, "middle-click after finish")
	out[0].Click(bar.Event{Button: bar.ButtonLeft})
	notifier.AssertNoUpdate(t, refreshCh, "left-click on finished module error")
}
