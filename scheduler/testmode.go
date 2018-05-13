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
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/notifier"
)

// TestMode sets test mode for all schedulers.
// In test mode schedulers do not fire automatically, and time
// does not pass at all, until NextTick() or Advance* is called.
func TestMode() {
	testMutex.Lock()
	defer testMutex.Unlock()
	New = testNew
	Now = testNow
	// Set to non-zero time when entering test mode so that any IsZero
	// checks don't unexpectedly pass.
	nowInTest.Store(time.Date(2016, time.November, 25, 20, 47, 0, 0, time.UTC))
	// Clear the list of test schedulers so that TestMode() starts with
	// a clean slate.
	testSchedulers = make(schedulerList, 0)
}

// ExitTestMode exits test mode for all schedulers. Any schedulers created
// after this call will be real.
func ExitTestMode() {
	testMutex.Lock()
	defer testMutex.Unlock()
	New = realNew
	Now = time.Now
}

// testScheduler implements bar.Scheduler for test mode.
type testScheduler struct {
	sync.Mutex
	notifyFn     func()
	notifyCh     <-chan struct{}
	nextTrigger  time.Time
	interval     time.Duration
	paused       bool
	fireOnResume bool
}

func testNew() Controller {
	fn, ch := notifier.New()
	s := &testScheduler{notifyFn: fn, notifyCh: ch}
	testMutex.Lock()
	defer testMutex.Unlock()
	testSchedulers = append(testSchedulers, s)
	return s
}

func (s *testScheduler) setTrigger(next time.Time, interval time.Duration) {
	s.Lock()
	defer s.Unlock()
	s.nextTrigger = next
	s.interval = interval
}

func (s *testScheduler) Tick() <-chan struct{} {
	return s.notifyCh
}

func (s *testScheduler) At(when time.Time) bar.Scheduler {
	s.setTrigger(when, time.Duration(0))
	return s
}

func (s *testScheduler) After(delay time.Duration) bar.Scheduler {
	s.setTrigger(Now().Add(delay), time.Duration(0))
	return s
}

func (s *testScheduler) Every(interval time.Duration) bar.Scheduler {
	s.setTrigger(Now(), interval)
	return s
}

func (s *testScheduler) Stop() {
	s.setTrigger(time.Time{}, time.Duration(0))
}

func (s *testScheduler) Pause() {
	s.Lock()
	defer s.Unlock()
	s.paused = true
}

func (s *testScheduler) Resume() {
	s.Lock()
	defer s.Unlock()
	s.paused = false
	if s.fireOnResume {
		s.fireOnResume = false
		s.notifyFn()
	}
}

// tickAfter returns the next trigger time for the scheduler.
// This is used in test mode to determine the next firing scheduler
// and advance time to it.
func (s *testScheduler) tickAfter(now time.Time) time.Time {
	s.Lock()
	defer s.Unlock()
	if s.interval == time.Duration(0) {
		return s.nextTrigger
	}
	elapsedIntervals := now.Sub(s.nextTrigger) / s.interval
	return s.nextTrigger.Add(s.interval * (elapsedIntervals + 1))
}

var testMutex sync.Mutex

// nowInTest tracks the current time in test mode.
var nowInTest atomic.Value // of time.Time

func testNow() time.Time {
	return nowInTest.Load().(time.Time)
}

// schedulerList implements sort.Interface for schedulers based
// on their next trigger time.
type schedulerList []*testScheduler

func (l schedulerList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l schedulerList) Len() int      { return len(l) }
func (l schedulerList) Less(i, j int) bool {
	now := Now()
	return l[i].tickAfter(now).Before(l[j].tickAfter(now))
}

// testSchedulers tracks all schedulers created in test mode.
var testSchedulers schedulerList

// sortedSchedulers returns a list of test schedulers sorted by the
// time remaining until their next tick.
func sortedSchedulers() []*testScheduler {
	testMutex.Lock()
	defer testMutex.Unlock()
	sort.Sort(testSchedulers)
	return testSchedulers
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
			s.Lock()
			if s.paused {
				s.fireOnResume = true
			} else {
				s.notifyFn()
			}
			s.Unlock()
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
