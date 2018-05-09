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

package scheduler

import (
	"testing"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/stretchrcom/testify/assert"
)

func assertTriggered(t *testing.T, s bar.Scheduler, message string) {
	select {
	case <-s.Tick():
	case <-time.After(time.Second):
		assert.Fail(t, "scheduler did not trigger", message)
	}
}

func assertNotTriggered(t *testing.T, s bar.Scheduler, message string) {
	select {
	case <-s.Tick():
		assert.Fail(t, "scheduler was triggered", message)
	case <-time.After(10 * time.Millisecond):
	}
}

func TestStop(t *testing.T) {
	ExitTestMode()

	sch := New()
	assertNotTriggered(t, sch, "when not scheduled")

	sch.After(5 * time.Millisecond)
	sch.Stop()
	assertNotTriggered(t, sch, "when stopped")

	sch.Every(5 * time.Millisecond)
	sch.Stop()
	assertNotTriggered(t, sch, "when stopped")

	sch.At(Now().Add(5 * time.Millisecond))
	sch.Stop()
	assertNotTriggered(t, sch, "when stopped")

	sch.After(10 * time.Millisecond)
	assertTriggered(t, sch, "after interval elapses")

	sch.Stop()
	assertNotTriggered(t, sch, "when elapsed scheduler is stopped")

	sch.Stop()
	assertNotTriggered(t, sch, "when elapsed scheduler is stopped again")
}

func TestPauseResume(t *testing.T) {
	ExitTestMode()
	sch := New()

	sch.At(Now().Add(5 * time.Millisecond))
	sch.Pause()
	assertNotTriggered(t, sch, "when paused")

	sch.Resume()
	assertTriggered(t, sch, "when resumed")

	sch.Resume()
	assertNotTriggered(t, sch, "repeated resume is nop")
}

func TestRepeating(t *testing.T) {
	ExitTestMode()
	sch := New()

	sch.Every(5 * time.Millisecond)
	assertTriggered(t, sch, "after interval elapses")
	assertTriggered(t, sch, "after interval elapses")
	assertTriggered(t, sch, "after interval elapses")

	sch.Pause()
	assertNotTriggered(t, sch, "when paused")
	time.Sleep(15 * time.Millisecond) // > 2 intervals.

	sch.Resume()
	assertTriggered(t, sch, "when resumed")

	sch.Stop()
	assertNotTriggered(t, sch, "only once on resume")

	sch.After(5 * time.Millisecond)
	assertTriggered(t, sch, "after delay elapses")
	assertNotTriggered(t, sch, "after first trigger")
}

func TestCoalescedUpdates(t *testing.T) {
	ExitTestMode()
	sch := New()
	sch.Every(15 * time.Millisecond)
	time.Sleep(31 * time.Millisecond)
	assertTriggered(t, sch, "after multiple intervals")
	assertNotTriggered(t, sch, "multiple updates coalesced")
}
