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

// Package fail provides methods to test and verify failing assertions.
package fail

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Failed returns true if the given function failed the test.
func Failed(fn func(*testing.T)) bool {
	fakeT := &testing.T{}
	doneCh := make(chan bool)
	// Need a separate goroutine in case the test function calls FailNow.
	go func() {
		defer func() { doneCh <- true }()
		fn(fakeT)
	}()
	<-doneCh
	return fakeT.Failed()
}

// AssertFails asserts that the given test function fails the test.
func AssertFails(t *testing.T, fn func(*testing.T), formatAndArgs ...interface{}) {
	if !Failed(fn) {
		require.Fail(t, "Expected test to fail", formatAndArgs...)
	}
}
