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
Package timing provides a testable interface for timing and scheduling.

This makes it simple to update a module at a fixed interval or
at a fixed point in time.

Typically, modules will make a scheduler:
    mod.sch = timing.NewScheduler()
and use the scheduling calls to control the update timing:
    mod.sch.Every(time.Second)

The Stream() goroutine will then loop over the ticker, and update
the module with fresh information:
    for range mod.sch.Tick() {
	  // update code.
    }

This will automatically suspend processing when the bar is hidden.

Modules should also use timing.Now() instead of time.Now() to control time
during tests.
*/
package timing

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/soumya92/barista/notifier"
)

// scheduler implements timing.Scheduler using a timer for fixed delays
// and a ticker for fixed intervals.
type scheduler struct {
	timer  *time.Timer
	ticker *time.Ticker

	mutex        sync.Mutex
	notifyFn     func()
	notifyCh     <-chan struct{}
	paused       bool
	fireOnResume bool
}

var (
	// Keeps track of all schedulers, to allow timing.Pause and timing.Resume.
	schedulers   []Scheduler
	schedulersMu sync.RWMutex
)

// Whether new schedulers are created paused.
var paused atomic.Value // of bool

// NewScheduler creates a new scheduler.
func NewScheduler() Scheduler {
	s := schedulerMaker()
	if p, ok := paused.Load().(bool); ok && p {
		s.pause()
	}
	schedulersMu.Lock()
	defer schedulersMu.Unlock()
	schedulers = append(schedulers, s)
	return s
}

// Pause timing.
func Pause() {
	paused.Store(true)
	schedulersMu.RLock()
	defer schedulersMu.RUnlock()
	for _, sch := range schedulers {
		sch.pause()
	}
}

// Resume timing.
func Resume() {
	paused.Store(false)
	schedulersMu.RLock()
	defer schedulersMu.RUnlock()
	for _, sch := range schedulers {
		sch.resume()
	}
}

// schedulerMaker creates a scheduler, replaced in test mode.
var schedulerMaker = newScheduler

// newScheduler returns a new real scheduler.
func newScheduler() Scheduler {
	fn, ch := notifier.New()
	return &scheduler{notifyFn: fn, notifyCh: ch}
}

func (s *scheduler) maybeTick() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.paused {
		s.fireOnResume = true
	} else {
		s.notifyFn()
	}
}

func (s *scheduler) Tick() <-chan struct{} {
	return s.notifyCh
}

func (s *scheduler) At(when time.Time) Scheduler {
	return s.After(when.Sub(Now()))
}

func (s *scheduler) After(delay time.Duration) Scheduler {
	s.Stop()
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.timer = time.AfterFunc(delay, s.maybeTick)
	return s
}

func (s *scheduler) Every(interval time.Duration) Scheduler {
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

func (s *scheduler) pause() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paused = true
}

func (s *scheduler) resume() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paused = false
	if s.fireOnResume {
		s.fireOnResume = false
		s.notifyFn()
	}
}
