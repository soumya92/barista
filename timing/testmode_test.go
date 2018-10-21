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

package timing

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"barista.run/testing/notifier"
	"github.com/stretchr/testify/require"
)

func TestTiming_TestMode(t *testing.T) {
	TestMode()

	sch1 := NewScheduler()
	sch2 := NewScheduler()
	sch3 := NewScheduler()

	startTime := Now()
	require.Equal(t, startTime, NextTick(),
		"next tick doesn't change time when nothing is scheduled")
	notifier.AssertNoUpdate(t, sch1.Tick(), "when not scheduled")
	notifier.AssertNoUpdate(t, sch2.Tick(), "when not scheduled")
	notifier.AssertNoUpdate(t, sch3.Tick(), "when not scheduled")

	sch1.After(time.Hour)
	sch2.After(time.Second)
	sch3.After(time.Minute)

	require.Equal(t, startTime.Add(time.Second), NextTick(),
		"triggers earliest scheduler")
	notifier.AssertNotified(t, sch2.Tick(), "triggers earliest scheduler")
	notifier.AssertNoUpdate(t, sch1.Tick(), "only earliest scheduler triggers")
	notifier.AssertNoUpdate(t, sch3.Tick(), "only earliest scheduler triggers")

	require.Equal(t, startTime.Add(time.Minute), NextTick(),
		"triggers next scheduler")
	notifier.AssertNoUpdate(t, sch2.Tick(), "already elapsed")
	notifier.AssertNotified(t, sch3.Tick(), "earliest scheduler triggers")
	notifier.AssertNoUpdate(t, sch1.Tick(), "not yet elapsed")

	AdvanceBy(20 * time.Minute)
	notifier.AssertNoUpdate(t, sch2.Tick(), "already elapsed")
	notifier.AssertNoUpdate(t, sch3.Tick(), "already elapsed")
	notifier.AssertNoUpdate(t, sch1.Tick(), "did not advance far enough")

	AdvanceBy(2 * time.Hour)
	notifier.AssertNoUpdate(t, sch2.Tick(), "already elapsed")
	notifier.AssertNoUpdate(t, sch3.Tick(), "already elapsed")
	notifier.AssertNotified(t, sch1.Tick(), "when advancing beyond trigger duration")

	now := Now()
	sch1.At(now.Add(time.Minute))
	sch1.At(now.Add(time.Hour))
	sch1.At(now.Add(time.Second))
	require.Equal(t, now.Add(time.Second), NextTick())
	notifier.AssertNotified(t, sch1.Tick())
}

func TestRepeating_TestMode(t *testing.T) {
	TestMode()
	sch1 := NewScheduler()
	sch2 := NewScheduler()
	now := Now()

	sch1.Every(time.Minute)
	sch2.Every(10 * time.Minute)
	for i := 1; i < 10; i++ {
		require.Equal(t,
			now.Add(time.Duration(i)*time.Minute),
			NextTick(),
			"repeated scheduler")
		notifier.AssertNotified(t, sch1.Tick(), "repeated scheduler")
	}
	require.Equal(t,
		now.Add(time.Duration(10)*time.Minute),
		NextTick(), "at overlap")
	notifier.AssertNotified(t, sch1.Tick(), "at overlap")
	notifier.AssertNotified(t, sch2.Tick(), "at overlap")

	now = Now()
	sch1.Stop()
	sch2.Stop()
	require.Equal(t, now, NextTick(), "no ticks when stopped")
}

func TestRepeatingChange_TestMode(t *testing.T) {
	TestMode()
	sch := NewScheduler()
	now := Now()

	sch.Every(time.Minute)
	require.Equal(t, now.Add(1*time.Minute), NextTick())
	require.Equal(t, now.Add(2*time.Minute), NextTick())
	require.Equal(t, now.Add(3*time.Minute), NextTick())

	now = now.Add(3 * time.Minute)
	sch.Every(time.Hour)
	require.Equal(t, now.Add(1*time.Hour), NextTick())
	require.Equal(t, now.Add(2*time.Hour), NextTick())
	require.Equal(t, now.Add(3*time.Hour), NextTick())
}

func TestMultipleTriggers_TestMode(t *testing.T) {
	TestMode()
	sch1 := NewScheduler()
	sch2 := NewScheduler()
	sch3 := NewScheduler()
	now := Now()

	sch1.Every(time.Minute)
	sch2.After(time.Minute)
	sch3.At(Now().Add(time.Minute))
	require.Equal(t, now.Add(time.Minute), NextTick(), "multiple triggers")
	notifier.AssertNotified(t, sch1.Tick(), "multiple triggers")
	notifier.AssertNotified(t, sch2.Tick(), "multiple triggers")
	notifier.AssertNotified(t, sch3.Tick(), "multiple triggers")

	AdvanceBy(59*time.Second + 999*time.Millisecond)
	notifier.AssertNoUpdate(t, sch1.Tick(), "before interval elapses")

	AdvanceBy(10 * time.Millisecond)
	notifier.AssertNotified(t, sch1.Tick(), "after interval elapses")
}

func TestAdvanceWithRepeated_TestMode(t *testing.T) {
	TestMode()

	sch := NewScheduler()
	sch.Every(time.Second)

	var tickCount int32
	var launched sync.WaitGroup
	for i := 0; i < 60; i++ {
		launched.Add(1)
		// Ensure that no writes to sch's ticker will block,
		// by adding listeners to the channel in advance.
		go func() {
			launched.Done()
			<-sch.Tick()
			atomic.AddInt32(&tickCount, 1)
		}()
	}

	launched.Wait() // ensure goroutines are launched.
	AdvanceBy(time.Minute)

	// Expect all ticks to be received within 10ms of real time.
	time.Sleep(10 * time.Millisecond)
	actualTicks := atomic.LoadInt32(&tickCount)
	if actualTicks < 54 {
		require.Fail(t, "Not enough notifications",
			"Expected >= 54 ticks out of 60, only got %d tick(s)",
			actualTicks)
	}
}

func TestCoalescedUpdates_TestMode(t *testing.T) {
	TestMode()

	sch := NewScheduler()
	sch.Every(15 * time.Millisecond)
	AdvanceBy(45 * time.Second)
	runtime.Gosched()
	notifier.AssertNotified(t, sch.Tick(), "after multiple intervals")
	notifier.AssertNoUpdate(t, sch.Tick(), "multiple updates coalesced")
}

func TestPauseResume_TestMode(t *testing.T) {
	TestMode()

	sch := NewScheduler()
	start := Now()
	expected := start

	Pause()
	sch.Every(time.Minute)
	sch2 := NewScheduler().Every(time.Minute)

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with paused scheduler")
	notifier.AssertNoUpdate(t, sch.Tick(), "while paused")
	notifier.AssertNoUpdate(t, sch2.Tick(), "created while paused")

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with paused scheduler")
	notifier.AssertNoUpdate(t, sch.Tick(), "while paused")
	notifier.AssertNoUpdate(t, sch2.Tick(), "while paused")

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with paused scheduler")
	notifier.AssertNoUpdate(t, sch.Tick(), "while paused")

	AdvanceBy(30 * time.Second)
	notifier.AssertNoUpdate(t, sch.Tick(), "while paused")

	Resume()
	notifier.AssertNotified(t, sch.Tick(), "when resumed")
	notifier.AssertNotified(t, sch2.Tick(), "when resumed")
	notifier.AssertNoUpdate(t, sch.Tick(), "only once when resumed")
	notifier.AssertNoUpdate(t, sch2.Tick(), "only once when resumed")

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with resumed scheduler")
	notifier.AssertNotified(t, sch.Tick(), "tick after resuming")
	notifier.AssertNotified(t, sch2.Tick(), "tick after resuming")
}

func TestPastTriggers_TestMode(t *testing.T) {
	TestMode()
	sch := NewScheduler()
	now := Now()
	sch.After(-1 * time.Minute)
	require.Equal(t, now, NextTick())
	notifier.AssertNotified(t, sch.Tick(), "negative delay notifies immediately")
	sch.At(Now().Add(-1 * time.Minute))
	require.Equal(t, now, NextTick())
	notifier.AssertNotified(t, sch.Tick(), "past trigger notifies immediately")

	Pause()
	sch.After(-1 * time.Minute)
	NextTick()
	notifier.AssertNoUpdate(t, sch.Tick(), "when paused")
	Resume()
	NextTick()
	notifier.AssertNotified(t, sch.Tick(), "on resume")

	Pause()
	sch.At(Now().Add(-1 * time.Minute))
	NextTick()
	notifier.AssertNoUpdate(t, sch.Tick(), "when paused")
	Resume()
	NextTick()
	notifier.AssertNotified(t, sch.Tick(), "on resume")

	require.Panics(t, func() {
		sch.Every(-1 * time.Second)
	}, "negative repeating interval")
}

func TestTestModeReset(t *testing.T) {
	TestMode()
	sch1 := NewScheduler().Every(time.Second)

	startTime := Now()
	require.Equal(t, startTime.Add(time.Second), NextTick())
	notifier.AssertNotified(t, sch1.Tick(), "triggers every second")

	require.Equal(t, startTime.Add(2*time.Second), NextTick())
	notifier.AssertNotified(t, sch1.Tick(), "triggers every second")

	Pause()
	require.Equal(t, startTime.Add(3*time.Second), NextTick())
	notifier.AssertNoUpdate(t, sch1.Tick(), "when paused")

	TestMode()
	sch2 := NewScheduler().Every(time.Minute)

	startTime = Now()
	require.Equal(t, startTime.Add(time.Minute), NextTick())
	notifier.AssertNoUpdate(t, sch1.Tick(), "previous scheduler is not triggered")
	notifier.AssertNotified(t, sch2.Tick(), "new scheduler is triggered")

	require.Equal(t, startTime.Add(2*time.Minute), NextTick())
	notifier.AssertNoUpdate(t, sch1.Tick(), "previous scheduler is not triggered")
	notifier.AssertNotified(t, sch2.Tick(), "new scheduler is repeatedly triggered")
}
