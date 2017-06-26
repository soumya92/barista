// Copyright 2017 Google Inc.
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

/*
Package scheduler provides a testable interface for scheduling tasks.

This makes it simple to update a module at a fixed interval or
at a fixed point in time.

Typical usage would be:
    scheduler.Do(module.Update).At(someTime)

or keeping track of the scheduler to update it later:
    sch := scheduler.Do(module.Update).Every(time.Minute)
    // change the scheduler to run every second instead.
    sch.Every(time.Second)
*/
package scheduler

import (
	"sort"
	"sync"
	"time"
)

// Scheduler represents a function triggered on a schedule.
// It provides an interface to stop or modify the trigger schedule.
type Scheduler interface {
	// At sets the scheduler to trigger a specific time.
	// This will replace any pending triggers.
	At(time.Time) Scheduler

	// After sets the scheduler to trigger after a delay.
	// This will replace any pending triggers.
	After(time.Duration) Scheduler

	// Every sets the scheduler to trigger at an interval.
	// This will replace any pending triggers.
	Every(time.Duration) Scheduler

	// Stop cancels all further triggers for the scheduler.
	Stop()
}

// scheduler holds either a timer or a ticker that triggers
// the given function.
type scheduler struct {
	timer  *time.Timer
	ticker *time.Ticker
	do     func()
	mutex  sync.Mutex
	// For test mode, keep track of the next triggers.
	nextTrigger time.Time
	interval    time.Duration
}

// Do creates a scheduler that calls the given function when triggered.
func Do(f func()) Scheduler {
	s := &scheduler{do: f}
	if testMode {
		testSchedulers = append(testSchedulers, s)
	}
	return s
}

func (s *scheduler) At(when time.Time) Scheduler {
	return s.After(when.Sub(Now()))
}

func (s *scheduler) After(delay time.Duration) Scheduler {
	s.Stop()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if testMode {
		s.nextTrigger = Now().Add(delay)
		return s
	}
	s.timer = time.AfterFunc(delay, s.do)
	return s
}

func (s *scheduler) Every(interval time.Duration) Scheduler {
	s.Stop()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if testMode {
		s.nextTrigger = Now()
		s.interval = interval
		return s
	}
	s.ticker = time.NewTicker(interval)
	go func() {
		s.mutex.Lock()
		ticker := s.ticker
		s.mutex.Unlock()
		if ticker == nil {
			// Scheduler stopped before goroutine was started.
			return
		}
		for range ticker.C {
			s.do()
		}
	}()
	return s
}

func (s *scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if testMode {
		s.nextTrigger = time.Time{}
		s.interval = time.Duration(0)
		return
	}
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

// tickAfter returns the next trigger time for the scheduler.
// This is used in test mode to determine the next firing scheduler
// and advance time to it.
func (s *scheduler) tickAfter(now time.Time) time.Time {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.interval == time.Duration(0) {
		return s.nextTrigger
	}
	elapsedIntervals := now.Sub(s.nextTrigger) / s.interval
	return s.nextTrigger.Add(s.interval * (elapsedIntervals + 1))
}

// Now returns the current time. Used for testing.
func Now() time.Time {
	if testMode {
		nowMutex.Lock()
		defer nowMutex.Unlock()
		return nowInTest
	}
	return time.Now()
}

// TestMode sets test mode for all schedulers.
// In test mode schedulers do not fire automatically, and time
// does not pass at all, until NextTick() or Advance* is called.
func TestMode(enabled bool) {
	nowMutex.Lock()
	defer nowMutex.Unlock()
	testMode = enabled
	nowInTest = time.Time{}
}

// testMode tracks whether all schedulers are in test mode.
var testMode = false

// nowInTest tracks the current time in test mode.
var nowInTest time.Time
var nowMutex sync.Mutex

// schedulerList implements sort.Interface for schedulers based
// on their next trigger time.
type schedulerList []*scheduler

func (l schedulerList) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l schedulerList) Len() int      { return len(l) }
func (l schedulerList) Less(i, j int) bool {
	now := Now()
	return l[i].tickAfter(now).Before(l[j].tickAfter(now))
}

// testSchedulers tracks all schedulers created in test mode.
var testSchedulers schedulerList

// NextTick triggers the next scheduler and returns the trigger time.
// It also advances test time to match.
func NextTick() time.Time {
	sort.Sort(testSchedulers)
	now := Now()
	for _, s := range testSchedulers {
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
	sort.Sort(testSchedulers)
	now := Now()
	for _, s := range testSchedulers {
		nextTick := s.tickAfter(now)
		if nextTick.After(now) && !nextTick.After(newTime) {
			setNowTo(nextTick)
			go s.do()
		}
	}
	setNowTo(newTime)
}

// setNowTo sets nowInTest but ensures that data access is guarded
// by the mutex.
func setNowTo(now time.Time) {
	nowMutex.Lock()
	defer nowMutex.Unlock()
	nowInTest = now
}
