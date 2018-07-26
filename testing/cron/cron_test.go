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

package cron

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	// Not waiting 31 seconds for this test.
	waits = []int{0, 1, 2, 3}
}

func failNTimes(n int) func(*testing.T) {
	count := 0
	return func(t *testing.T) {
		count++
		if count > n {
			return
		}
		assert.Fail(t, "Failing N Times", "%d out of %d", count, n)
	}
}

func mockGetenv(eventType string) func(string) string {
	return func(key string) string {
		if key == "TRAVIS_EVENT_TYPE" {
			return eventType
		}
		return os.Getenv(key)
	}
}

func TestNotCron(t *testing.T) {
	getenv = mockGetenv("not-cron")
	Test(t, func(*testing.T) {
		assert.Fail(t, "test func called but not a cron build")
	})
}

func TestCron(t *testing.T) {
	getenv = mockGetenv("cron")

	testT := &testing.T{}
	start := time.Now()
	Test(testT, failNTimes(100))
	end := time.Now()
	if !testT.Failed() {
		assert.Fail(t, "Expected Test to fail")
	}
	assert.WithinDuration(t, start.Add(6*time.Second), end, time.Second)

	testT = &testing.T{}
	start = time.Now()
	Test(testT, failNTimes(2))
	end = time.Now()
	if testT.Failed() {
		assert.Fail(t, "Expected Test to pass after retries")
	}
	assert.WithinDuration(t, start.Add(1*time.Second), end, time.Second,
		"Test should only wait 0+1 seconds (for the 2 failures)")
}
