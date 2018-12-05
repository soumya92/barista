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

package localtz

import (
	"io/ioutil"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"

	"barista.run/testing/notifier"

	"github.com/stretchr/testify/require"
)

func newTempFile() string {
	dir, err := ioutil.TempDir("", "localtz")
	if err != nil {
		panic(err)
	}
	return path.Join(dir, "localtime")
}

// Must run before localtz.go init(), so use IIFE instead.
var ignored = func() int {
	tzFile = ""
	return 0
}()

func TestTimezoneChanges(t *testing.T) {
	require := require.New(t)
	atomic.StoreUint32(&testMode, 0)
	tzFile = newTempFile()
	go watchTz(tzFile)

	next := Next()
	os.Symlink("/usr/share/zoneinfo/Europe/Berlin", tzFile)
	notifier.AssertClosed(t, next)

	berlin, _ := time.LoadLocation("Europe/Berlin")
	require.Equal(berlin, Get())

	next = Next()
	os.Remove(tzFile)
	notifier.AssertNoUpdate(t, next, "On unlink")
	require.Equal(berlin, Get(), "timezone does not revert to local")

	os.Symlink("/usr/share/zoneinfo/Africa/Kinshasa", tzFile)
	notifier.AssertClosed(t, next)

	westCongo, _ := time.LoadLocation("Africa/Kinshasa")
	require.Equal(westCongo, Get())
}

func TestErrorNotSymlink(t *testing.T) {
	require := require.New(t)
	atomic.StoreUint32(&testMode, 0)
	tzFile = newTempFile()
	go watchTz(tzFile)

	next := Next()
	os.OpenFile(tzFile, os.O_RDONLY|os.O_CREATE, 0666)
	notifier.AssertClosed(t, next, "not a symlink")
	require.Equal(time.Local, Get(), "reverts to time.Local")
}

func TestErrorBadLocation(t *testing.T) {
	require := require.New(t)
	atomic.StoreUint32(&testMode, 0)
	tzFile = newTempFile()
	go watchTz(tzFile)

	next := Next()
	os.Symlink("/usr/share/zoneinfo/Nowhere/SomeCity", tzFile)
	notifier.AssertClosed(t, next, "bad location")
	require.Equal(time.Local, Get(), "reverts to time.Local")
}

func TestErrorBadSymlink(t *testing.T) {
	require := require.New(t)
	atomic.StoreUint32(&testMode, 0)
	tzFile = newTempFile()
	go watchTz(tzFile)

	next := Next()
	os.Remove(tzFile)
	os.Symlink("foobar", tzFile)
	notifier.AssertClosed(t, next, "bad symlink") // 23!
	require.Equal(time.Local, Get(), "reverts to time.Local")
}

func TestPermanentError(t *testing.T) {
	require := require.New(t)
	atomic.StoreUint32(&testMode, 0)
	tzFile = newTempFile()
	go watchTz(tzFile)

	os.OpenFile(tzFile, os.O_RDONLY|os.O_CREATE, 0666)
	atomic.AddInt32(&errCount, 3) // exhaust retries.
	time.Sleep(2 * time.Second)   // for the last loop to expire.
	require.Equal(time.Local, Get(), "resets to Local on permanent failure")

	os.Remove(tzFile)
	next := Next()
	os.Symlink("/usr/share/zoneinfo/America/Mexico_City", tzFile)
	require.Equal(time.Local, Get(), "fixed on Local after permanent failure")
	notifier.AssertNoUpdate(t, next, "Not notified after permanent failure")
}

func TestSetForTest(t *testing.T) {
	require := require.New(t)
	atomic.StoreUint32(&testMode, 0)
	tzFile = newTempFile()
	go watchTz(tzFile)

	SetForTest(time.Local)
	require.Equal(time.Local, Get())

	next := Next()
	SetForTest(time.UTC)
	notifier.AssertClosed(t, next, "on test mode set")
	require.Equal(time.UTC, Get())

	loc, _ := time.LoadLocation("Asia/Tokyo")
	SetForTest(loc)
	require.Equal(loc, Get())

	SetForTest(nil)
	require.Nil(Get())

	next = Next()
	os.Remove(tzFile)
	os.Symlink("/usr/share/zoneinfo/Europe/Rome", tzFile)
	notifier.AssertNoUpdate(t, next, "Real changes ignored in test mode")
}
