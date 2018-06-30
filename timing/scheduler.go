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
	"sync"
	"sync/atomic"
	"time"

	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/notifier"
)

// Scheduler represents a trigger that can be repeating or one-off, and
// is intrinsically tied to the running bar. This means that if the trigger
// condition occurs while the bar is paused, it will not fire until the bar
// is next resumed, making it ideal for scheduling work that should only be
// performed while the bar is active.
type Scheduler struct {
	// for real scheduling.
	timer  *time.Timer
	ticker *time.Ticker

	// for test scheduling.
	nextTrigger time.Time
	interval    time.Duration

	mutex        sync.Mutex
	notifyFn     func()
	notifyCh     <-chan struct{}
	paused       bool
	fireOnResume bool
}

var (
	// Keeps track of all schedulers, for timing.Pause/Resume and test mode.
	schedulers   []*Scheduler
	schedulersMu sync.RWMutex
)

// Whether new schedulers are created paused.
var paused atomic.Value // of bool
func init() {
	paused.Store(false)
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	fn, ch := notifier.New()
	s := &Scheduler{notifyFn: fn, notifyCh: ch}
	l.Attach(s, ch, "")
	if paused.Load().(bool) {
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

// Tick returns a channel that receives an empty value
// when the scheduler is triggered.
func (s *Scheduler) Tick() <-chan struct{} {
	return s.notifyCh
}

// At sets the scheduler to trigger a specific time.
// This will replace any pending triggers.
func (s *Scheduler) At(when time.Time) *Scheduler {
	l.Fine("%s At(%v)", l.ID(s), when)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if inTestMode() {
		s.nextTrigger = when
		s.interval = time.Duration(0)
		return s
	}
	s.stop()
	s.timer = time.AfterFunc(when.Sub(Now()), s.maybeTrigger)
	return s
}

// After sets the scheduler to trigger after a delay.
// This will replace any pending triggers.
func (s *Scheduler) After(delay time.Duration) *Scheduler {
	l.Fine("%s After(%v)", l.ID(s), delay)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if inTestMode() {
		s.nextTrigger = Now().Add(delay)
		s.interval = time.Duration(0)
		return s
	}
	s.stop()
	s.timer = time.AfterFunc(delay, s.maybeTrigger)
	return s
}

// Every sets the scheduler to trigger at an interval.
// This will replace any pending triggers.
func (s *Scheduler) Every(interval time.Duration) *Scheduler {
	l.Fine("%s Every(%v)", l.ID(s), interval)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if inTestMode() {
		s.nextTrigger = Now()
		s.interval = interval
		return s
	}
	s.stop()
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
			s.maybeTrigger()
		}
	}()
	return s
}

// Stop cancels all further triggers for the scheduler.
func (s *Scheduler) Stop() {
	l.Fine("%s Stop", l.ID(s))
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if inTestMode() {
		s.nextTrigger = time.Time{}
		s.interval = time.Duration(0)
		return
	}
	s.stop()
}

func (s *Scheduler) maybeTrigger() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.paused {
		s.fireOnResume = true
	} else {
		s.notifyFn()
	}
}

func (s *Scheduler) stop() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

func (s *Scheduler) pause() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paused = true
}

func (s *Scheduler) resume() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.paused = false
	if s.fireOnResume {
		s.fireOnResume = false
		s.notifyFn()
	}
}
