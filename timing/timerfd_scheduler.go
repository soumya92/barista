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

// +build linux

package timing

import (
	"errors"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	l "barista.run/logging"
	"barista.run/timing/internal/timerfd"
)

var _ schedulerImpl = &timerfdScheduler{}

// timerfdScheduler represents a scheduler backed by system real-time clock.
type timerfdScheduler struct {
	mu       sync.Mutex
	timerfd  *timerfd.Timerfd
	interval time.Duration
	offset   time.Duration
	f        func()
}

// NewRealtimeScheduler creates a scheduler backed by system real-time clock.
//
// It properly handles system suspend (sleep mode) and time adjustments. For periodic timers,
// it triggers immediately whenever time changes discontinuously. For one-shot timers
// (At and After), it will fire immediately if the time is skipped over
// the set trigger time, and will properly wait for it otherwise.
//
// This scheduler is only properly supported on Linux. On other systems,
// plain scheduler based on "time" package is returned.
//
// In order to clean up resources associated with it,
// remember to call Stop().
func NewRealtimeScheduler() (*Scheduler, error) {
	if testModeScheduler := maybeNewTestModeScheduler(); testModeScheduler != nil {
		return newScheduler(testModeScheduler), nil
	}

	timerfd, err := timerfd.NewRealtimeTimerfd()
	if err != nil {
		return nil, err
	}
	impl := &timerfdScheduler{timerfd: timerfd}
	go impl.loop()

	return newScheduler(impl), nil
}

// At implements the schedulerImpl interface.
func (s *timerfdScheduler) At(when time.Time, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// note for future: only valid for CLOCK_REALTIME
	timespec, err := unix.TimeToTimespec(when)
	if err != nil {
		panic("TimeToTimespec failed: " + err.Error())
	}

	err = s.timerfd.Settime(&unix.ItimerSpec{
		Value: timespec,
	}, nil, true, false)
	if err != nil {
		panic("Settime failed: " + err.Error())
	}

	s.interval = 0
	s.f = f
}

// After implements the schedulerImpl interface.
func (s *timerfdScheduler) After(delay time.Duration, f func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.timerfd.Settime(&unix.ItimerSpec{
		Value: unix.NsecToTimespec(delay.Nanoseconds()),
	}, nil, false, false)
	if err != nil {
		panic("Settime failed: " + err.Error())
	}

	s.interval = 0
	s.f = f
}

// Every implements the schedulerImpl interface.
func (s *timerfdScheduler) Every(interval time.Duration, f func()) {
	now := Now()
	offset := now.Sub(now.Truncate(interval))
	s.EveryAlign(interval, offset, f)
}

// EveryAlign implements the schedulerImpl interface.
func (s *timerfdScheduler) EveryAlign(interval time.Duration, offset time.Duration, f func()) {
	if interval <= 0 {
		panic(errors.New("non-positive interval for RealtimeScheduler#Every"))
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.interval = interval
	s.offset = offset - offset.Truncate(interval)
	s.f = f

	s.rearmPeriodicTimerLocked()
}

// Stop implements the schedulerImpl interface.
func (s *timerfdScheduler) Stop() {
	s.timerfd.Settime(&unix.ItimerSpec{}, nil, false, false)
}

// Close implements the schedulerImpl interface.
func (s *timerfdScheduler) Close() {
	s.timerfd.Close()
}

// rearmPeriodicTimerLocked panics when an error occurs,
// because any errors are not handleable runtime failures,
// but rather extreme conditions. See comments near panics below.
func (s *timerfdScheduler) rearmPeriodicTimerLocked() {
	if s.interval == 0 {
		return
	}

	var currentTimespec unix.Timespec
	if err := unix.ClockGettime(int32(s.timerfd.GetClockid()), &currentTimespec); err != nil {
		// see man clock_gettime: should never happens in sane conditions
		panic("rearmPeriodicTimer failed: " + err.Error())
	}
	currentTime := time.Unix(currentTimespec.Unix())
	initial := nextAlignedExpiration(currentTime, s.interval, s.offset)
	initialTimespec, err := unix.TimeToTimespec(initial)
	if err != nil {
		// can only happen on systems having 32-bit timespec when
		// (roughly) currentTime+interval+offset overflows,
		// i.e. at the moment just before having system-wide year 2038 apocalypse,
		// or when the user sets insanely long timer period.
		panic("rearmPeriodicTimer failed: " + err.Error())
	}
	err = s.timerfd.Settime(&unix.ItimerSpec{
		Interval: unix.NsecToTimespec(s.interval.Nanoseconds()),
		Value:    initialTimespec,
	}, nil, true, true)
	if err != nil {
		// see man timerfd
		panic("rearmPeriodicTimer failed: " + err.Error())
	}
}

func (s *timerfdScheduler) loop() {
	// note that s.f is called without holding mutex: it doesn't need it,
	// and in more general case it should be allowed to call timer methods
	// without causing a deadlock
	for {
		_, err := s.timerfd.Wait()
		if err != nil {
			if err == timerfd.ErrTimerfdCancelled {
				l.Fine("%s: errTimerfdCancelled (discontinuous time change detected)", l.ID(s))
				s.mu.Lock()
				f := s.f
				s.rearmPeriodicTimerLocked()
				s.mu.Unlock()
				f()
			} else if err == os.ErrClosed {
				// Close has been called
				return
			} else {
				l.Log("%s: timerfd.Wait() returned %v", l.ID(s), err)
				return
			}
		} else {
			s.mu.Lock()
			f := s.f
			s.mu.Unlock()
			f()
		}
	}
}
