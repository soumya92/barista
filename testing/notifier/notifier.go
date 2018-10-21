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

// Package notifier provides assertions that notifier channels (<-chan struct{})
// received or did not receive a signal.
package notifier // import "barista.run/testing/notifier"

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var positiveTimeout = 10 * time.Millisecond
var negativeTimeout = time.Second

// AssertNotified asserts that the given channel received a notification.
func AssertNotified(t *testing.T, ch <-chan struct{}, formatAndArgs ...interface{}) {
	select {
	case _, ok := <-ch:
		if !ok {
			require.Fail(t, "Expected notification but channel was closed", formatAndArgs...)
		}
	case <-time.After(negativeTimeout):
		require.Fail(t, "Expected notification not received", formatAndArgs...)
	}
}

// AssertClosed asserts that the given channel was closed.
func AssertClosed(t *testing.T, ch <-chan struct{}, formatAndArgs ...interface{}) {
	select {
	case _, ok := <-ch:
		if ok {
			require.Fail(t, "Expected channel close, received notification", formatAndArgs...)
		}
	case <-time.After(negativeTimeout):
		require.Fail(t, "Channel not closed when expected", formatAndArgs...)
	}
}

// AssertNoUpdate asserts that the given channel was not notified or closed.
func AssertNoUpdate(t *testing.T, ch <-chan struct{}, formatAndArgs ...interface{}) {
	select {
	case <-ch:
		require.Fail(t, "Unexpected notification", formatAndArgs...)
	case <-time.After(positiveTimeout):
	}
}
