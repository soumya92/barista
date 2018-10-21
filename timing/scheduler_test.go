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
	notifier.AssertNoUpdate(t, sch.Tick(), "when not scheduled")

	sch.After(50 * time.Millisecond).Stop()
	notifier.AssertNoUpdate(t, sch.Tick(), "when stopped")

	sch.Every(50 * time.Millisecond).Stop()
	notifier.AssertNoUpdate(t, sch.Tick(), "when stopped")

	sch.At(Now().Add(50 * time.Millisecond)).Stop()
	notifier.AssertNoUpdate(t, sch.Tick(), "when stopped")

	sch.After(10 * time.Millisecond)
	notifier.AssertNotified(t, sch.Tick(), "after interval elapses")

	sch.Stop()
	notifier.AssertNoUpdate(t, sch.Tick(), "when elapsed scheduler is stopped")

	sch.Stop()
	notifier.AssertNoUpdate(t, sch.Tick(), "when elapsed scheduler is stopped again")
}

func TestPauseResume(t *testing.T) {
	ExitTestMode()
	sch := NewScheduler()

	sch.At(Now().Add(5 * time.Millisecond))
	Pause()
	schWhilePaused := NewScheduler().After(2 * time.Millisecond)

	notifier.AssertNoUpdate(t, sch.Tick(), "when paused")
	notifier.AssertNoUpdate(t, schWhilePaused.Tick(), "scheduler created while paused")

	Resume()
	notifier.AssertNotified(t, sch.Tick(), "when resumed")
	notifier.AssertNotified(t, schWhilePaused.Tick(), "when resumed")

	Resume()
	notifier.AssertNoUpdate(t, sch.Tick(), "repeated resume is nop")
}

func TestRepeating(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real repeating test in short mode")
	}
	ExitTestMode()
	sch := NewScheduler()

	sch.Every(100 * time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.Tick(), "after interval elapses")
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.Tick(), "after interval elapses")
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNotified(t, sch.Tick(), "after interval elapses")

	Pause()
	time.Sleep(100 * time.Millisecond)
	notifier.AssertNoUpdate(t, sch.Tick(), "when paused")
	time.Sleep(1 * time.Second) // > 2 intervals.
	Resume()

	notifier.AssertNotified(t, sch.Tick(), "when resumed")
	notifier.AssertNoUpdate(t, sch.Tick(), "only once on resume")

	sch.After(20 * time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	notifier.AssertNotified(t, sch.Tick(), "after delay elapses")
	notifier.AssertNoUpdate(t, sch.Tick(), "after first trigger")
}

func TestCoalescedUpdates(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real coalescing test in short mode")
	}
	ExitTestMode()
	sch := NewScheduler()
	sch.Every(300 * time.Millisecond)
	time.Sleep(3100 * time.Millisecond)
	notifier.AssertNotified(t, sch.Tick(), "after multiple intervals")
	notifier.AssertNoUpdate(t, sch.Tick(), "multiple updates coalesced")
}

func TestPastTriggers(t *testing.T) {
	ExitTestMode()
	sch := NewScheduler()
	sch.After(-1 * time.Minute)
	notifier.AssertNotified(t, sch.Tick(), "negative delay notifies immediately")
	sch.At(Now().Add(-1 * time.Minute))
	notifier.AssertNotified(t, sch.Tick(), "past trigger notifies immediately")

	Pause()
	sch.After(-1 * time.Minute)
	notifier.AssertNoUpdate(t, sch.Tick(), "when paused")
	Resume()
	notifier.AssertNotified(t, sch.Tick(), "on resume")

	Pause()
	sch.At(Now().Add(-1 * time.Minute))
	notifier.AssertNoUpdate(t, sch.Tick(), "when paused")
	Resume()
	notifier.AssertNotified(t, sch.Tick(), "on resume")

	require.Panics(t, func() {
		sch.Every(-1 * time.Second)
	}, "negative repeating interval")
}
