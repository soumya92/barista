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

	"github.com/soumya92/barista/testing/notifier"
	"github.com/stretchr/testify/require"
)

func TestTiming_TestMode(t *testing.T) {
	TestMode()
	defer ExitTestMode()

	sch1 := NewScheduler()
	sch2 := NewScheduler()
	sch3 := NewScheduler()

	startTime := Now()
	require.Equal(t, startTime, NextTick(),
		"next tick doesn't change time when nothing is scheduled")
	notifier.AssertNoUpdate(t, sch1.C, "when not scheduled")
	notifier.AssertNoUpdate(t, sch2.C, "when not scheduled")
	notifier.AssertNoUpdate(t, sch3.C, "when not scheduled")

	sch1.After(time.Hour)
	sch2.After(time.Second)
	sch3.After(time.Minute)

	require.Equal(t, startTime.Add(time.Second), NextTick(),
		"triggers earliest scheduler")
	notifier.AssertNotified(t, sch2.C, "triggers earliest scheduler")
	notifier.AssertNoUpdate(t, sch1.C, "only earliest scheduler triggers")
	notifier.AssertNoUpdate(t, sch3.C, "only earliest scheduler triggers")

	require.Equal(t, startTime.Add(time.Minute), NextTick(),
		"triggers next scheduler")
	notifier.AssertNoUpdate(t, sch2.C, "already elapsed")
	notifier.AssertNotified(t, sch3.C, "earliest scheduler triggers")
	notifier.AssertNoUpdate(t, sch1.C, "not yet elapsed")

	AdvanceBy(20 * time.Minute)
	notifier.AssertNoUpdate(t, sch2.C, "already elapsed")
	notifier.AssertNoUpdate(t, sch3.C, "already elapsed")
	notifier.AssertNoUpdate(t, sch1.C, "did not advance far enough")

	AdvanceBy(2 * time.Hour)
	notifier.AssertNoUpdate(t, sch2.C, "already elapsed")
	notifier.AssertNoUpdate(t, sch3.C, "already elapsed")
	notifier.AssertNotified(t, sch1.C, "when advancing beyond trigger duration")

	now := Now()
	sch1.At(now.Add(time.Minute))
	sch1.At(now.Add(time.Hour))
	sch1.At(now.Add(time.Second))
	require.Equal(t, now.Add(time.Second), NextTick())
	notifier.AssertNotified(t, sch1.C)
}

func TestRepeating_TestMode(t *testing.T) {
	TestMode()
	defer ExitTestMode()
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
		notifier.AssertNotified(t, sch1.C, "repeated scheduler")
	}
	require.Equal(t,
		now.Add(time.Duration(10)*time.Minute),
		NextTick(), "at overlap")
	notifier.AssertNotified(t, sch1.C, "at overlap")
	notifier.AssertNotified(t, sch2.C, "at overlap")

	now = Now()
	sch1.Stop()
	sch2.Stop()
	require.Equal(t, now, NextTick(), "no ticks when stopped")
}

func TestRepeatingChange_TestMode(t *testing.T) {
	TestMode()
	defer ExitTestMode()
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
	defer ExitTestMode()
	sch1 := NewScheduler()
	sch2 := NewScheduler()
	sch3 := NewScheduler()
	now := Now()

	sch1.Every(time.Minute)
	sch2.After(time.Minute)
	sch3.At(Now().Add(time.Minute))
	require.Equal(t, now.Add(time.Minute), NextTick(), "multiple triggers")
	notifier.AssertNotified(t, sch1.C, "multiple triggers")
	notifier.AssertNotified(t, sch2.C, "multiple triggers")
	notifier.AssertNotified(t, sch3.C, "multiple triggers")

	AdvanceBy(59*time.Second + 999*time.Millisecond)
	notifier.AssertNoUpdate(t, sch1.C, "before interval elapses")

	AdvanceBy(10 * time.Millisecond)
	notifier.AssertNotified(t, sch1.C, "after interval elapses")
}

func TestAdvanceWithRepeated_TestMode(t *testing.T) {
	TestMode()
	defer ExitTestMode()

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
			<-sch.C
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
	defer ExitTestMode()

	sch := NewScheduler()
	sch.Every(15 * time.Millisecond)
	AdvanceBy(45 * time.Second)
	runtime.Gosched()
	notifier.AssertNotified(t, sch.C, "after multiple intervals")
	notifier.AssertNoUpdate(t, sch.C, "multiple updates coalesced")
}

func TestPauseResume_TestMode(t *testing.T) {
	TestMode()
	defer ExitTestMode()

	sch := NewScheduler()
	start := Now()
	expected := start

	Pause()
	sch.Every(time.Minute)
	sch2 := NewScheduler().Every(time.Minute)

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with paused scheduler")
	notifier.AssertNoUpdate(t, sch.C, "while paused")
	notifier.AssertNoUpdate(t, sch2.C, "created while paused")

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with paused scheduler")
	notifier.AssertNoUpdate(t, sch.C, "while paused")
	notifier.AssertNoUpdate(t, sch2.C, "while paused")

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with paused scheduler")
	notifier.AssertNoUpdate(t, sch.C, "while paused")

	AdvanceBy(30 * time.Second)
	notifier.AssertNoUpdate(t, sch.C, "while paused")

	Resume()
	notifier.AssertNotified(t, sch.C, "when resumed")
	notifier.AssertNotified(t, sch2.C, "when resumed")
	notifier.AssertNoUpdate(t, sch.C, "only once when resumed")
	notifier.AssertNoUpdate(t, sch2.C, "only once when resumed")

	expected = expected.Add(time.Minute)
	require.Equal(t, expected, NextTick(), "with resumed scheduler")
	notifier.AssertNotified(t, sch.C, "tick after resuming")
	notifier.AssertNotified(t, sch2.C, "tick after resuming")
}

func TestPastTriggers_TestMode(t *testing.T) {
	TestMode()
	defer ExitTestMode()
	sch := NewScheduler()
	now := Now()
	sch.After(-1 * time.Minute)
	require.Equal(t, now, NextTick())
	notifier.AssertNotified(t, sch.C, "negative delay notifies immediately")
	sch.At(Now().Add(-1 * time.Minute))
	require.Equal(t, now, NextTick())
	notifier.AssertNotified(t, sch.C, "past trigger notifies immediately")

	Pause()
	sch.After(-1 * time.Minute)
	NextTick()
	notifier.AssertNoUpdate(t, sch.C, "when paused")
	Resume()
	NextTick()
	notifier.AssertNotified(t, sch.C, "on resume")

	Pause()
	sch.At(Now().Add(-1 * time.Minute))
	NextTick()
	notifier.AssertNoUpdate(t, sch.C, "when paused")
	Resume()
	NextTick()
	notifier.AssertNotified(t, sch.C, "on resume")

	require.Panics(t, func() {
		sch.Every(-1 * time.Second)
	}, "negative repeating interval")
}

func TestTestModeReset(t *testing.T) {
	TestMode()
	defer ExitTestMode()
	sch1 := NewScheduler().Every(time.Second)

	startTime := Now()
	require.Equal(t, startTime.Add(time.Second), NextTick())
	notifier.AssertNotified(t, sch1.C, "triggers every second")

	require.Equal(t, startTime.Add(2*time.Second), NextTick())
	notifier.AssertNotified(t, sch1.C, "triggers every second")

	Pause()
	require.Equal(t, startTime.Add(3*time.Second), NextTick())
	notifier.AssertNoUpdate(t, sch1.C, "when paused")

	TestMode()
	sch2 := NewScheduler().Every(time.Minute)

	startTime = Now()
	require.Equal(t, startTime.Add(time.Minute), NextTick())
	notifier.AssertNoUpdate(t, sch1.C, "previous scheduler is not triggered")
	notifier.AssertNotified(t, sch2.C, "new scheduler is triggered")

	require.Equal(t, startTime.Add(2*time.Minute), NextTick())
	notifier.AssertNoUpdate(t, sch1.C, "previous scheduler is not triggered")
	notifier.AssertNotified(t, sch2.C, "new scheduler is repeatedly triggered")

	sch1.After(time.Millisecond)
	require.Equal(t, startTime.Add(3*time.Minute), NextTick())
	notifier.AssertNoUpdate(t, sch1.C, "previous scheduler is not triggered")
	notifier.AssertNotified(t, sch2.C, "new scheduler is repeatedly triggered")

	TestMode()
	sch1 = NewScheduler().After(time.Minute)
	sch2 = NewScheduler().Every(2 * time.Minute)

	TestMode()
	AdvanceBy(time.Hour)

	notifier.AssertNoUpdate(t, sch1.C, "previous scheduler is not triggered")
	notifier.AssertNoUpdate(t, sch2.C, "previous scheduler is not triggered")
}
