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

/*
Package cron provides a function to run a test only in Travis CI cron
runs, and retry the test with increasing delays a few times before
failing the build.

The primary purpose of this method is to allow cron test that are
non-hermetic and run against live (usually http) endpoints. Since the
live endpoints could occasionally throw errors, there is built-in retry
with delays between attempts.
*/
package cron // import "barista.run/testing/cron"

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var getenv = os.Getenv
var waits = []int{1, 3, 7, 15}

// Test runs a test if running in the CI's cron mode. It handles retries if the
// test returns an error, but passes through failures to the test suite. This
// allows the test function to retry by returning transient errors, while not
// wasting attempts on non-retryable failures.
func Test(t *testing.T, testFunc func() error) {
	if evt := getenv("TRAVIS_EVENT_TYPE"); evt != "cron" {
		t.Skipf("Skipping LiveVersion test for event type '%s'", evt)
	}
	for _, wait := range waits {
		err := testFunc()
		if err == nil {
			return
		}
		t.Logf("Waiting %ds due to %v", wait, err)
		time.Sleep(time.Duration(wait) * time.Second)
	}
	require.NoError(t, testFunc(), "On last cron attempt")
}
