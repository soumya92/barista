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
	"testing"
	"time"

	"barista.run/testing/notifier"
	"github.com/stretchr/testify/require"
)

func TestStop(t *testing.T) {
	ExitTestMode()

	sch := NewScheduler()
	notifier.AssertNoUpdate(t, sch.C, "when not scheduled")

	sch.After(50 * time.Millisecond).Stop()
	notifier.AssertNoUpdate(t, sch.C, "when stopped")

	sch.Every(50 * time.Millisecond).Stop()
	notifier.AssertNoUpdate(t, sch.C, "when stopped")

	sch.At(Now().Add(50 * time.Millisecond)).Stop()
	notifier.AssertNoUpdate(t, sch.C, "when stopped")

	sch.After(10 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")

	sch.Stop()
	notifier.AssertNoUpdate(t, sch.C, "when elapsed scheduler is stopped")

	sch.Stop()
	notifier.AssertNoUpdate(t, sch.C, "when elapsed scheduler is stopped again")
}

func TestPauseResume(t *testing.T) {
	ExitTestMode()
	sch := NewScheduler()

	sch.At(Now().Add(5 * time.Millisecond))
	Pause()
	schWhilePaused := NewScheduler().After(2 * time.Millisecond)

	notifier.AssertNoUpdate(t, sch.C, "when paused")
	notifier.AssertNoUpdate(t, schWhilePaused.C, "scheduler created while paused")

	Resume()
	notifier.AssertNotified(t, sch.C, "when resumed")
	notifier.AssertNotified(t, schWhilePaused.C, "when resumed")

	Resume()
	notifier.AssertNoUpdate(t, sch.C, "repeated resume is nop")
}

func TestRepeating(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real repeating test in short mode")
	}
	ExitTestMode()
	sch := NewScheduler()

	sch.Every(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")

	Pause()
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNoUpdate(t, sch.C, "when paused")
	time.Sleep(1 * time.Second) // > 2 intervals.
	Resume()

	notifier.AssertNotified(t, sch.C, "when resumed")
	notifier.AssertNoUpdate(t, sch.C, "only once on resume")

	sch.After(20 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after delay elapses")
	notifier.AssertNoUpdate(t, sch.C, "after first trigger")
}

func TestCoalescedUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real coalescing test in short mode")
	}
	ExitTestMode()
	sch := NewScheduler()
	sch.Every(300 * time.Millisecond)
	time.Sleep(3100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after multiple intervals")
	notifier.AssertNoUpdate(t, sch.C, "multiple updates coalesced")
}

func TestPastTriggers(t *testing.T) {
	ExitTestMode()
	sch := NewScheduler()
	sch.After(-1 * time.Minute)
	notifier.AssertNotified(t, sch.C, "negative delay notifies immediately")
	sch.At(Now().Add(-1 * time.Minute))
	notifier.AssertNotified(t, sch.C, "past trigger notifies immediately")

	Pause()
	sch.After(-1 * time.Minute)
	notifier.AssertNoUpdate(t, sch.C, "when paused")
	Resume()
	notifier.AssertNotified(t, sch.C, "on resume")

	Pause()
	sch.At(Now().Add(-1 * time.Minute))
	notifier.AssertNoUpdate(t, sch.C, "when paused")
	Resume()
	notifier.AssertNotified(t, sch.C, "on resume")

	require.Panics(t, func() {
		sch.Every(-1 * time.Second)
	}, "negative repeating interval")
}

func TestTick(t *testing.T) {
	ExitTestMode()
	now := Now()
	timeChan := make(chan time.Time, 1)
	go func() {
		NewScheduler().After(3 * time.Second).Tick()
		timeChan <- Now()
	}()
	require.WithinDuration(t,
		now.Add(3*time.Second), <-timeChan,
		50*time.Millisecond, "Tick waits for expected duration")
}

func TestReplaceInterval(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real repeating test in short mode")
	}
	ExitTestMode()
	sch := NewScheduler()

	sch.Every(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.C, "after interval elapses")
	Pause()
	sch.Every(1 * time.Second)
	Resume()
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNoUpdate(t, sch.C, "after interval elapses")
	time.Sleep(1 * time.Second)
	notifier.AssertNotified(t, sch.C, "after interval elapses")
}
