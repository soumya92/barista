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

	"github.com/soumya92/barista/base/notifier"
	l "github.com/soumya92/barista/logging"
)

// scheduler implements a Scheduler tied to actual time.
type scheduler struct {
	sync.Mutex

	timer   *time.Timer
	ticker  *time.Ticker
	quitter chan struct{}

	notifyFn func()
	notifyCh <-chan struct{}

	waiting int32 // basically bool, but we need atomics.
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
func NewScheduler() Scheduler {
	fn, ch := notifier.New()
	s := &scheduler{notifyFn: fn, notifyCh: ch}
	l.Attach(s, ch, "")
	if testMode {
		t := &testScheduler{scheduler: s}
		l.Attach(t, s, "")
		return t
	}
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

// Tick returns a channel that receives an empty value
// when the scheduler is triggered.
func (s *scheduler) Tick() <-chan struct{} {
	return s.notifyCh
}

func (s *scheduler) At(when time.Time) Scheduler {
	l.Fine("%s At(%v)", l.ID(s), when)
	s.Lock()
	defer s.Unlock()
	s.stop()
	s.timer = time.AfterFunc(when.Sub(Now()), s.maybeTrigger)
	return s
}

func (s *scheduler) After(delay time.Duration) Scheduler {
	l.Fine("%s After(%v)", l.ID(s), delay)
	s.Lock()
	defer s.Unlock()
	s.stop()
	s.timer = time.AfterFunc(delay, s.maybeTrigger)
	return s
}

func (s *scheduler) Every(interval time.Duration) Scheduler {
	l.Fine("%s Every(%v)", l.ID(s), interval)
	if interval <= 0 {
		panic(errors.New("non-positive interval for Scheduler#Every"))
	}
	s.Lock()
	defer s.Unlock()
	s.stop()
	s.quitter = make(chan struct{})
	s.ticker = time.NewTicker(interval)
	go func() {
		s.Lock()
		ticker := s.ticker
		quitter := s.quitter
		s.Unlock()
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

func (s *scheduler) Stop() {
	l.Fine("%s Stop", l.ID(s))
	s.Lock()
	defer s.Unlock()
	s.stop()
}

func (s *scheduler) maybeTrigger() {
	if !atomic.CompareAndSwapInt32(&s.waiting, 0, 1) {
		return
	}
	await(func() {
		if atomic.CompareAndSwapInt32(&s.waiting, 1, 0) {
			s.notifyFn()
		}
	})
}

func (s *scheduler) stop() {
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
