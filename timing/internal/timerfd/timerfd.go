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

package timerfd

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// ErrTimerfdCancelled is the error returned when timer is cancelled due to
// discontinuous clock change. See man timerfd_settime for details.
var ErrTimerfdCancelled = errors.New("Timerfd was cancelled due to discontinuous change")

// Timerfd provides higher-level abstraction for Linux-specific timerfd timers.
type Timerfd struct {
	fd      *os.File
	clockid int
}

func newTimerfd(clockid int) (*Timerfd, error) {
	fd, err := unix.TimerfdCreate(clockid, unix.TFD_NONBLOCK|unix.TFD_CLOEXEC)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), "Timerfd")
	return &Timerfd{fd: f, clockid: clockid}, nil
}

// Settime arms or disarms the timer. See man timerfd_settime for details.
func (t *Timerfd) Settime(newValue *unix.ItimerSpec, oldValue *unix.ItimerSpec, absolute bool, cancelOnSet bool) error {
	rawConn, err := t.fd.SyscallConn()
	if err != nil {
		return err
	}
	var err2 error
	err = rawConn.Control(func(fd uintptr) {
		var flags int
		if absolute {
			flags |= unix.TFD_TIMER_ABSTIME
		}
		if cancelOnSet {
			flags |= unix.TFD_TIMER_CANCEL_ON_SET
		}
		err2 = unix.TimerfdSettime(int(fd), flags, newValue, oldValue)
	})
	if err != nil {
		return err
	}
	return err2
}

// Wait waits until timer expiration. See man timerfd_create for details.
func (t *Timerfd) Wait() (expirations uint64, err error) {
	var buf [8]byte

	n, err := t.fd.Read(buf[:])
	if err != nil {
		if pe, ok := err.(*os.PathError); ok {
			err = pe.Err
		}
		if err == unix.ECANCELED {
			err = ErrTimerfdCancelled
		}
		return 0, err
	}
	if n != 8 {
		panic(fmt.Sprintf("Timerfd returned %d bytes (expected 8)", n))
	}
	return nativeEndian.Uint64(buf[:]), nil
}

// GetClockid returns the clock id that this timerfd uses.
func (t *Timerfd) GetClockid() int {
	return t.clockid
}

// Close closes the underlying timerfd descriptor.
func (t *Timerfd) Close() error {
	return t.fd.Close()
}

// NewRealtimeTimerfd returns a Timerfd backed by CLOCK_REALTIME.
func NewRealtimeTimerfd() (*Timerfd, error) {
	return newTimerfd(unix.CLOCK_REALTIME)
}
