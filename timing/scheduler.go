// Copyright 2020 Google Inc.
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

	notifyFn func()
	waiting  int32 // basically bool, but we need atomics.

	schedulerImpl schedulerImpl
}

var (
	// A set of channels to be closed by timing.Resume.
	// This allows schedulers to wait for resume, without
	// requiring a reference to each created scheduler.
	waiters []chan struct{}
	paused  = false

	mu sync.Mutex
)

type schedulerImpl interface {
	At(time.Time, func())
	After(time.Duration, func())
	Every(time.Duration, func())
	EveryAlign(time.Duration, time.Duration, func())
	Stop()
	Close()
}

func newScheduler(impl schedulerImpl) *Scheduler {
	s := new(Scheduler)
	s.schedulerImpl = impl
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
	l.Fine("%s At(%v)", l.ID(s), when)
	s.schedulerImpl.At(when, s.maybeTrigger)
	return s
}

// After sets the scheduler to trigger after a delay.
// This will replace any pending triggers.
func (s *Scheduler) After(delay time.Duration) *Scheduler {
	l.Fine("%s After(%v)", l.ID(s), delay)
	s.schedulerImpl.After(delay, s.maybeTrigger)
	return s
}

// Every sets the scheduler to trigger at an interval.
// This will replace any pending triggers.
func (s *Scheduler) Every(interval time.Duration) *Scheduler {
	if interval <= 0 {
		panic(errors.New("non-positive interval for Scheduler#Every"))
	}
	l.Fine("%s Every(%v)", l.ID(s), interval)
	s.schedulerImpl.Every(interval, s.maybeTrigger)
	return s
}

// EveryAlign sets the scheduler to trigger at an interval.
//
// Offset specifies the scheduler alignment. For example, if interval=1min,
// and offset=11s, the timer will trigger every minute at exactly :11 seconds
// of the underlying clock. This makes most sense for schedulers based on
// the real time clock.
// Usually offset should be zero. A clock that displays the time with minute
// precision should probably update at :00 seconds, and
// interval=1min and offset=0 do exactly that.
//
// This will replace any pending triggers.
func (s *Scheduler) EveryAlign(interval time.Duration, offset time.Duration) *Scheduler {
	if interval <= 0 {
		panic(errors.New("non-positive interval for Scheduler#EveryAlign"))
	}
	if offset < 0 {
		panic(errors.New("negative offset for Scheduler#EveryAlign"))
	}
	l.Fine("%s EveryAlign(%v, %v)", l.ID(s), interval, offset)
	s.schedulerImpl.EveryAlign(interval, offset, s.maybeTrigger)
	return s
}

// Stop cancels all further triggers for the scheduler.
func (s *Scheduler) Stop() {
	l.Fine("%s Stop", l.ID(s))
	s.schedulerImpl.Stop()
}

// Close cleans up all resources allocated by the scheduler, if necessary.
func (s *Scheduler) Close() {
	l.Fine("%s Close", l.ID(s))
	s.schedulerImpl.Close()
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
