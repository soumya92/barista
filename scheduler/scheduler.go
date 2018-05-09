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

/*
Package scheduler provides a testable interface for scheduling tasks.

This makes it simple to update a module at a fixed interval or
at a fixed point in time.

Typically, modules will make a scheduler:
    mod.sch = scheduler.New()
and use the scheduling calls to control the update timing:
    mod.sch.Every(time.Second)

The Stream() goroutine will then loop over the ticker, and update
the module with fresh information:
    for range mod.sch.Tick() {
	  // update code.
    }

This will automatically suspend processing when the bar is hidden.
*/
package scheduler

import (
	"sync"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/notifier"
)

// Controller extends a Scheduler with pause/resume/stop.
// This allows a Controller to be provided as just a Scheduler
// when only the scheduling capabilities are required.
type Controller interface {
	bar.Scheduler

	// Pause suspends calling the Do function until resumed.
	// The scheduler will still tick as normal (if repeating),
	// but multiple ticks will still only result in a single
	// call to the Do function when resumed.
	Pause()

	// Resume restores the scheduler's ticking behaviour,
	// and calls Do function if it was originally scheduled to
	// be called while the scheduler was paused.
	Resume()
}

// scheduler implements bar.Scheduler using a timer for fixed delays
// and a ticker for fixed intervals.
type scheduler struct {
	bar.Notifier
	timer        *time.Timer
	ticker       *time.Ticker
	mutex        sync.Mutex
	paused       bool
	fireOnResume bool
}

// New creates a new scheduler.
var New = realNew

// Now returns the current time.
var Now = time.Now

// realNew returns a real scheduler.
func realNew() Controller {
	return &scheduler{Notifier: notifier.New()}
}

func (s *scheduler) maybeTick() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.paused {
		s.fireOnResume = true
	} else {
		s.Notifier.Notify()
	}
}

func (s *scheduler) At(when time.Time) bar.Scheduler {
	return s.After(when.Sub(Now()))
}

func (s *scheduler) After(delay time.Duration) bar.Scheduler {
	s.Stop()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.timer = time.AfterFunc(delay, s.maybeTick)
	return s
}

func (s *scheduler) Every(interval time.Duration) bar.Scheduler {
	s.Stop()
	s.mutex.Lock()
	defer s.mutex.Unlock()
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
			s.maybeTick()
		}
	}()
	return s
}

func (s *scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

func (s *scheduler) Pause() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paused = true
}

func (s *scheduler) Resume() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paused = false
	if s.fireOnResume {
		s.fireOnResume = false
		s.Notifier.Notify()
	}
}
