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
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"barista.run/base/notifier"
	l "barista.run/logging"
)

// Scheduler represents a trigger that can be repeating or one-off, and
// is intrinsically tied to the running bar. This means that if the trigger
// condition occurs while the bar is paused, it will not fire until the bar
// is next resumed, making it ideal for scheduling work that should only be
// performed while the bar is active.
type Scheduler struct {
	// A channel that receives an empty struct for each tick of the scheduler.
	C <-chan struct{}

	mu      sync.Mutex
	timer   *time.Timer
	ticker  *time.Ticker
	quitter chan struct{}

	notifyFn func()
	waiting  int32 // basically bool, but we need atomics.

	// For test mode
	testMode  bool
	startTime time.Time
	interval  time.Duration
}

var (
	// A set of channels to be closed by timing.Resume.
	// This allows schedulers to wait for resume, without
	// requiring a reference to each created scheduler.
	waiters  []chan struct{}
	paused   = false
	testMode = false

	mu sync.Mutex
)

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	mu.Lock()
	s := &Scheduler{testMode: testMode}
	mu.Unlock()
	s.notifyFn, s.C = notifier.New()
	l.Register(s, "C")
	return s
}

// Pause timing.
func Pause() {
	mu.Lock()
	defer mu.Unlock()
	paused = true
}

// await executes the given function when the bar is running.
// If the bar is paused, it waits for the bar to resume.
func await(fn func()) {
	mu.Lock()
	if !paused {
		mu.Unlock()
		fn()
		return
	}
	ch := make(chan struct{})
	waiters = append(waiters, ch)
	mu.Unlock()
	go func() {
		<-ch
		fn()
	}()
}

// Resume timing.
func Resume() {
	mu.Lock()
	defer mu.Unlock()
	paused = false
	for _, ch := range waiters {
		close(ch)
	}
	waiters = nil
}

// Tick waits until the next tick of the scheduler.
// Equivalent to <-scheduler.C, but returns true to allow for sch.Tick() { ... }
func (s *Scheduler) Tick() bool {
	<-s.C
	return true
}

// At sets the scheduler to trigger a specific time.
// This will replace any pending triggers.
func (s *Scheduler) At(when time.Time) *Scheduler {
	if s.testMode {
		return s.testModeAt(when)
	}
	l.Fine("%s At(%v)", l.ID(s), when)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.timer = time.AfterFunc(when.Sub(Now()), s.maybeTrigger)
	return s
}

// After sets the scheduler to trigger after a delay.
// This will replace any pending triggers.
func (s *Scheduler) After(delay time.Duration) *Scheduler {
	if s.testMode {
		return s.testModeAfter(delay)
	}
	l.Fine("%s After(%v)", l.ID(s), delay)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.timer = time.AfterFunc(delay, s.maybeTrigger)
	return s
}

// Every sets the scheduler to trigger at an interval.
// This will replace any pending triggers.
func (s *Scheduler) Every(interval time.Duration) *Scheduler {
	if interval <= 0 {
		panic(errors.New("non-positive interval for Scheduler#Every"))
	}
	if s.testMode {
		return s.testModeEvery(interval)
	}
	l.Fine("%s Every(%v)", l.ID(s), interval)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
	s.quitter = make(chan struct{})
	s.ticker = time.NewTicker(interval)
	go func() {
		s.mu.Lock()
		ticker := s.ticker
		quitter := s.quitter
		s.mu.Unlock()
		if ticker == nil || quitter == nil {
			// Scheduler stopped before goroutine was started.
			return
		}
		for {
			select {
			case <-ticker.C:
				s.maybeTrigger()
			case <-quitter:
				return
			}
		}
	}()
	return s
}

// Stop cancels all further triggers for the scheduler.
func (s *Scheduler) Stop() {
	if s.testMode {
		s.testModeStop()
		return
	}
	l.Fine("%s Stop", l.ID(s))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stop()
}

func (s *Scheduler) maybeTrigger() {
	if !atomic.CompareAndSwapInt32(&s.waiting, 0, 1) {
		return
	}
	await(func() {
		if atomic.CompareAndSwapInt32(&s.waiting, 1, 0) {
			s.notifyFn()
		}
	})
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
	if s.quitter != nil {
		close(s.quitter)
		s.quitter = nil
	}
}
