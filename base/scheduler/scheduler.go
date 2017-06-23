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
	"time"
)

// Scheduler represents a function triggered on a schedule.
// It provides an interface to stop or modify the trigger schedule.
type Scheduler interface {
	At(time.Time) Scheduler
	After(time.Duration) Scheduler
	Every(time.Duration) Scheduler
	Stop()
}

// scheduler holds either a timer or a ticker that triggers
// the given function.
type scheduler struct {
	timer  *time.Timer
	ticker *time.Ticker
	do     func()
}

// Do creates a scheduler that calls the given function when triggered.
func Do(f func()) Scheduler {
	return &scheduler{do: f}
}

// At sets the scheduler to trigger a specific time.
// This will replace any pending triggers.
func (s *scheduler) At(when time.Time) Scheduler {
	return s.After(when.Sub(Now()))
}

// After sets the scheduler to trigger after a delay.
// This will replace any pending triggers.
func (s *scheduler) After(delay time.Duration) Scheduler {
	s.Stop()
	s.timer = time.AfterFunc(delay, s.do)
	return s
}

// Every sets the scheduler to trigger at an interval.
// This will replace any pending triggers.
func (s *scheduler) Every(interval time.Duration) Scheduler {
	s.Stop()
	s.ticker = time.NewTicker(interval)
	go func() {
		ticker := s.ticker
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

// Stop cancels all further triggers for the scheduler.
func (s *scheduler) Stop() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

// Now returns the current time. Used for testing.
var Now = time.Now
