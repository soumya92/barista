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
package cron

import (
	"os"
	"testing"
	"time"
)

var waits = []int{1, 3, 7, 15}

func Test(t *testing.T, testFunc func(t *testing.T)) {
	if evt := os.Getenv("TRAVIS_EVENT_TYPE"); evt != "cron" {
		t.Skipf("Skipping LiveVersion test for event type '%s'", evt)
	}
	for idx, wait := range waits {
		var testT *testing.T
		if idx == len(waits)-1 {
			// Final attempt runs on real testing.T, so the test
			// fails with any errors from testFunc.
			testT = t
		} else {
			testT = &testing.T{}
		}
		testFunc(testT)
		if !testT.Failed() {
			return
		}
		time.Sleep(time.Duration(wait) * time.Second)
	}
}
