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
package fail // import "barista.run/testing/fail"

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// failed returns true if the given function failed the test,
// using the provided fake testing.T instead of a new one, allowing
// re-use between calls.
func failed(fakeT *testing.T, fn func(*testing.T)) bool {
	doneCh := make(chan bool)
	// Need a separate goroutine in case the test function calls FailNow.
	go func() {
		defer func() { doneCh <- true }()
		fn(fakeT)
	}()
	<-doneCh
	return fakeT.Failed()
}

// Failed returns true if the given function failed the test.
func Failed(fn func(*testing.T)) bool {
	return failed(&testing.T{}, fn)
}

// AssertFails asserts that the given test function fails the test.
func AssertFails(t *testing.T, fn func(*testing.T), formatAndArgs ...interface{}) {
	if !Failed(fn) {
		require.Fail(t, "Expected test to fail", formatAndArgs...)
	}
}

// Setup represents an already set up test environment, which provides
// a variant on AssertFails that fails the test as normal, but also fails
// the test if the setup function causes test failures.
type TestSetup struct {
	setupFailed bool
	fakeT       *testing.T
}

// WithSetup shares the fake testing.T instance between a setup method
// and a test method, providing an AssertFails method that fails the test
// if the setup fails, or if the test method does not.
func Setup(setupFn func(*testing.T)) *TestSetup {
	t := &TestSetup{fakeT: &testing.T{}}
	t.setupFailed = failed(t.fakeT, setupFn)
	return t
}

// AssertFails asserts that the given test function fails the test, and
// that the setup function used did not cause any test failures.
func (s *TestSetup) AssertFails(t *testing.T, fn func(*testing.T), formatAndArgs ...interface{}) {
	if s.setupFailed {
		require.Fail(t, "Test failed in setup", formatAndArgs...)
	}
	if !failed(s.fakeT, fn) {
		require.Fail(t, "Expected test to fail", formatAndArgs...)
	}
}
