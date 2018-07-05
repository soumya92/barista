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

	"github.com/stretchrcom/testify/assert"
)

func TestTiming_TestMode(t *testing.T) {
	TestMode()

	sch1 := NewScheduler()
	sch2 := NewScheduler()
	sch3 := NewScheduler()

	startTime := Now()
	assert.Equal(t, startTime, NextTick(),
		"next tick doesn't change time when nothing is scheduled")
	assertNotTriggered(t, sch1, "when not scheduled")
	assertNotTriggered(t, sch2, "when not scheduled")
	assertNotTriggered(t, sch3, "when not scheduled")

	sch1.After(time.Hour)
	sch2.After(time.Second)
	sch3.After(time.Minute)

	assert.Equal(t, startTime.Add(time.Second), NextTick(),
		"triggers earliest scheduler")
	assertTriggered(t, sch2, "triggers earliest scheduler")
	assertNotTriggered(t, sch1, "only earliest scheduler triggers")
	assertNotTriggered(t, sch3, "only earliest scheduler triggers")

	assert.Equal(t, startTime.Add(time.Minute), NextTick(),
		"triggers next scheduler")
	assertNotTriggered(t, sch2, "already elapsed")
	assertTriggered(t, sch3, "earliest scheduler triggers")
	assertNotTriggered(t, sch1, "not yet elapsed")

	AdvanceBy(20 * time.Minute)
	assertNotTriggered(t, sch2, "already elapsed")
	assertNotTriggered(t, sch3, "already elapsed")
	assertNotTriggered(t, sch1, "did not advance far enough")

	AdvanceBy(2 * time.Hour)
	assertNotTriggered(t, sch2, "already elapsed")
	assertNotTriggered(t, sch3, "already elapsed")
	assertTriggered(t, sch1, "when advancing beyond trigger duration")

	now := Now()
	sch1.At(now.Add(time.Minute))
	sch1.At(now.Add(time.Hour))
	sch1.At(now.Add(time.Second))
	assert.Equal(t, now.Add(time.Second), NextTick())
	assertTriggered(t, sch1)
}

func TestRepeating_TestMode(t *testing.T) {
	TestMode()
	sch1 := NewScheduler()
	sch2 := NewScheduler()
	now := Now()

	sch1.Every(time.Minute)
	sch2.Every(10 * time.Minute)
	for i := 1; i < 10; i++ {
		assert.Equal(t,
			now.Add(time.Duration(i)*time.Minute),
			NextTick(),
			"repeated scheduler")
		assertTriggered(t, sch1, "repeated scheduler")
	}
	assert.Equal(t,
		now.Add(time.Duration(10)*time.Minute),
		NextTick(), "at overlap")
	assertTriggered(t, sch1, "at overlap")
	assertTriggered(t, sch2, "at overlap")

	now = Now()
	sch1.Stop()
	sch2.Stop()
	assert.Equal(t, now, NextTick(), "no ticks when stopped")
}

func TestRepeatingChange_TestMode(t *testing.T) {
	TestMode()
	sch := NewScheduler()
	now := Now()

	sch.Every(time.Minute)
	assert.Equal(t, now.Add(1*time.Minute), NextTick())
	assert.Equal(t, now.Add(2*time.Minute), NextTick())
	assert.Equal(t, now.Add(3*time.Minute), NextTick())

	now = now.Add(3 * time.Minute)
	sch.Every(time.Hour)
	assert.Equal(t, now.Add(1*time.Hour), NextTick())
	assert.Equal(t, now.Add(2*time.Hour), NextTick())
	assert.Equal(t, now.Add(3*time.Hour), NextTick())
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
	assert.Equal(t, now.Add(time.Minute), NextTick(), "multiple triggers")
	assertTriggered(t, sch1, "multiple triggers")
	assertTriggered(t, sch2, "multiple triggers")
	assertTriggered(t, sch3, "multiple triggers")

	AdvanceBy(59*time.Second + 999*time.Millisecond)
	assertNotTriggered(t, sch1, "before interval elapses")

	AdvanceBy(10 * time.Millisecond)
	assertTriggered(t, sch1, "after interval elapses")
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
		assert.Fail(t, "Not enough notifications",
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
	assertTriggered(t, sch, "after multiple intervals")
	assertNotTriggered(t, sch, "multiple updates coalesced")
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
	assert.Equal(t, expected, NextTick(), "with paused scheduler")
	assertNotTriggered(t, sch, "while paused")
	assertNotTriggered(t, sch2, "created while paused")

	expected = expected.Add(time.Minute)
	assert.Equal(t, expected, NextTick(), "with paused scheduler")
	assertNotTriggered(t, sch, "while paused")
	assertNotTriggered(t, sch2, "while paused")

	expected = expected.Add(time.Minute)
	assert.Equal(t, expected, NextTick(), "with paused scheduler")
	assertNotTriggered(t, sch, "while paused")

	AdvanceBy(30 * time.Second)
	assertNotTriggered(t, sch, "while paused")

	Resume()
	assertTriggered(t, sch, "when resumed")
	assertTriggered(t, sch2, "when resumed")
	assertNotTriggered(t, sch, "only once when resumed")
	assertNotTriggered(t, sch2, "only once when resumed")

	expected = expected.Add(time.Minute)
	assert.Equal(t, expected, NextTick(), "with resumed scheduler")
	assertTriggered(t, sch, "tick after resuming")
	assertTriggered(t, sch2, "tick after resuming")
}

func TestPastTriggers_TestMode(t *testing.T) {
	TestMode()
	sch := NewScheduler()
	now := Now()
	sch.After(-1 * time.Minute)
	assert.Equal(t, now, NextTick())
	assertTriggered(t, sch, "negative delay notifies immediately")
	sch.At(Now().Add(-1 * time.Minute))
	assert.Equal(t, now, NextTick())
	assertTriggered(t, sch, "past trigger notifies immediately")

	Pause()
	sch.After(-1 * time.Minute)
	NextTick()
	assertNotTriggered(t, sch, "when paused")
	Resume()
	NextTick()
	assertTriggered(t, sch, "on resume")

	Pause()
	sch.At(Now().Add(-1 * time.Minute))
	NextTick()
	assertNotTriggered(t, sch, "when paused")
	Resume()
	NextTick()
	assertTriggered(t, sch, "on resume")

	assert.Panics(t, func() {
		sch.Every(-1 * time.Second)
	}, "negative repeating interval")
}

func TestTestModeReset(t *testing.T) {
	TestMode()
	sch1 := NewScheduler().Every(time.Second)

	startTime := Now()
	assert.Equal(t, startTime.Add(time.Second), NextTick())
	assertTriggered(t, sch1, "triggers every second")

	assert.Equal(t, startTime.Add(2*time.Second), NextTick())
	assertTriggered(t, sch1, "triggers every second")

	Pause()
	assert.Equal(t, startTime.Add(3*time.Second), NextTick())
	assertNotTriggered(t, sch1, "when paused")

	TestMode()
	sch2 := NewScheduler().Every(time.Minute)

	startTime = Now()
	assert.Equal(t, startTime.Add(time.Minute), NextTick())
	assertNotTriggered(t, sch1, "previous scheduler is not triggered")
	assertTriggered(t, sch2, "new scheduler is triggered")

	assert.Equal(t, startTime.Add(2*time.Minute), NextTick())
	assertNotTriggered(t, sch1, "previous scheduler is not triggered")
	assertTriggered(t, sch2, "new scheduler is repeatedly triggered")
}
