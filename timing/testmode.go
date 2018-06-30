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
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	testMode  = false
	testMutex sync.RWMutex
)

func inTestMode() bool {
	testMutex.RLock()
	defer testMutex.RUnlock()
	return testMode
}

// nowInTest tracks the current time in test mode.
var nowInTest atomic.Value // of time.Time

func testNow() time.Time {
	return nowInTest.Load().(time.Time)
}

// TestMode sets test mode for all schedulers.
// In test mode schedulers do not fire automatically, and time
// does not pass at all, until NextTick() or Advance* is called.
func TestMode() {
	testMutex.Lock()
	schedulersMu.Lock()
	defer testMutex.Unlock()
	defer schedulersMu.Unlock()
	testMode = true
	Now = testNow
	schedulers = nil
	// Set to non-zero time when entering test mode so that any IsZero
	// checks don't unexpectedly pass.
	nowInTest.Store(time.Date(2016, time.November, 25, 20, 47, 0, 0, time.UTC))
	paused.Store(false)
}

// ExitTestMode exits test mode for all schedulers. Any schedulers created
// after this call will be real.
func ExitTestMode() {
	testMutex.Lock()
	schedulersMu.Lock()
	defer testMutex.Unlock()
	defer schedulersMu.Unlock()
	testMode = false
	Now = time.Now
	schedulers = nil
	paused.Store(false)
}

// tickAfter returns the next trigger time for the scheduler.
// This is used in test mode to determine the next firing scheduler
// and advance time to it.
func (s *Scheduler) tickAfter(now time.Time) time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.interval == time.Duration(0) {
		return s.nextTrigger
	}
	elapsedIntervals := now.Sub(s.nextTrigger) / s.interval
	return s.nextTrigger.Add(s.interval * (elapsedIntervals + 1))
}

// schedulerList implements sort.Interface for schedulers based
// on their next trigger time.
type schedulerList []*Scheduler

func (l schedulerList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l schedulerList) Len() int      { return len(l) }
func (l schedulerList) Less(i, j int) bool {
	now := Now()
	return l[i].tickAfter(now).Before(l[j].tickAfter(now))
}

// sortedSchedulers returns the list of schedulers sorted by the
// time remaining until their next tick.
func sortedSchedulers() []*Scheduler {
	schedulersMu.Lock()
	defer schedulersMu.Unlock()
	sort.Sort(schedulerList(schedulers))
	return schedulers
}

// NextTick triggers the next scheduler and returns the trigger time.
// It also advances test time to match.
func NextTick() time.Time {
	now := Now()
	for _, s := range sortedSchedulers() {
		nextTick := s.tickAfter(now)
		if nextTick.After(now) {
			AdvanceTo(nextTick)
			return nextTick
		}
	}
	return now
}

// AdvanceBy increments the test time by the given duration,
// and triggers any schedulers that were scheduled in the meantime.
func AdvanceBy(duration time.Duration) {
	AdvanceTo(Now().Add(duration))
}

// AdvanceTo increments the test time to the given time,
// and triggers any schedulers that were scheduled in the meantime.
func AdvanceTo(newTime time.Time) {
	now := Now()
	found := false
	for _, s := range sortedSchedulers() {
		nextTick := s.tickAfter(now)
		if nextTick.After(now) && !nextTick.After(newTime) {
			found = true
			nowInTest.Store(nextTick)
			s.maybeTrigger()
		}
	}
	if !found {
		nowInTest.Store(newTime)
		return
	}
	if Now() != newTime {
		AdvanceTo(newTime)
	}
}
