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
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/soumya92/barista/testing/fail"

	"github.com/stretchr/testify/require"
)

func init() {
	// Not waiting 31 seconds for this test.
	waits = []int{0, 1, 2, 3}
}

func failNTimes(n int) func() error {
	count := 0
	return func() error {
		count++
		if count > n {
			return nil
		}
		return fmt.Errorf("Failing N Times: %d out of %d", count, n)
	}
}

func mockGetenv(cronEnvVar string) func(string) string {
	return func(key string) string {
		if key == "CRON" {
			return cronEnvVar
		}
		return os.Getenv(key)
	}
}

func TestNotCron(t *testing.T) {
	getenv = mockGetenv("false")
	Test(t, func() error {
		require.Fail(t, "test func called but not a cron build")
		return nil
	})
}

func TestCron(t *testing.T) {
	getenv = mockGetenv("true")

	start := time.Now()
	fail.AssertFails(t, func(testT *testing.T) {
		Test(testT, failNTimes(100))
	}, "More than 4 failures from test function")
	end := time.Now()
	require.WithinDuration(t, start.Add(6*time.Second), end, time.Second)

	start = time.Now()
	Test(t, failNTimes(2))
	end = time.Now()
	require.WithinDuration(t, start.Add(1*time.Second), end, time.Second,
		"Test should only wait 0+1 seconds (for the 2 failures)")

	called := false
	Test(t, func() error {
		require.False(t, called, "test func called again after no error")
		called = true
		return nil
	})
}
