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

// Package localtz watches for changes to the local machine's time zone.
package localtz // import "barista.run/base/watchers/localtz"

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"barista.run/base/value"
	"barista.run/base/watchers/file"
	l "barista.run/logging"
)

// Overridden in tests.
var tzFile = "/etc/localtime"

var current value.Value // of *time.Location
var testMode uint32     // atomic bool

// Get returns the machine's current time zone.
func Get() *time.Location {
	return current.Get().(*time.Location)
}

// Next returns a channel that signals when the machine's time zone changes.
func Next() <-chan struct{} {
	return current.Next()
}

// SetForTest allows simulating a timezone change in tests.
func SetForTest(newZone *time.Location) {
	current.Set(newZone)
	atomic.StoreUint32(&testMode, 1)
}

func init() {
	current.Set(time.Local) // At startup, time.Local is correct.
	if tzFile != "" {
		go watchTz(tzFile)
	}
}

var errCount = int32(0)

func watchTz(tzFile string) {
	w := file.Watch(tzFile)
	defer w.Unsubscribe()

	for {
		err := watchTzOnce(w, tzFile)
		if err == errTestMode {
			return
		}
		l.Log("Timezone watcher exited: %v, falling back to time.Local", err)
		// fallback to time.Local on any errors.
		current.Set(time.Local)
		// throttle retries, in case the problem is transient.
		time.Sleep(time.Second)
		// limit retries. If three consecutive attempts fail, bail out.
		if atomic.AddInt32(&errCount, 1) > 3 {
			l.Log("Too many failures in timezone watcher, giving up")
			return
		}
	}
}

func watchTzOnce(w *file.Watcher, tzFile string) error {
	for {
		err := updateTz(tzFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		select {
		case <-w.Updates:
		case e := <-w.Errors:
			return e
		}
	}
}

var errTestMode = errors.New("TestMode")

func updateTz(tzFile string) error {
	if atomic.LoadUint32(&testMode) > 0 {
		return errTestMode
	}
	link, err := os.Readlink(tzFile)
	if err != nil {
		return err
	}
	l.Fine("Updating timezone from %s->%s", tzFile, link)
	ps := strings.Split(link, "/")
	len := len(ps)
	if len < 2 {
		return fmt.Errorf("Failed parsing zoneinfo: %s->%s", tzFile, link)
	}
	loc, err := time.LoadLocation(ps[len-2] + "/" + ps[len-1])
	if err != nil {
		return err
	}
	atomic.StoreInt32(&errCount, 0)
	current.Set(loc)
	l.Fine("Machine timezone changed to %v", loc)
	return nil
}
