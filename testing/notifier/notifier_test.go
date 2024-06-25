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

package notifier

import (
	"testing"
	"time"

	"github.com/soumya92/barista/base/notifier"
	"github.com/soumya92/barista/testing/fail"
)

func TestAssertions(t *testing.T) {
	fn, ch := notifier.New()

	AssertNoUpdate(t, ch, "no updates sent on channel")
	fn()
	AssertNotified(t, ch, "after update sent on channel")

	sendCh := make(chan struct{})
	ch = sendCh
	AssertNoUpdate(t, ch, "on open channel")
	close(sendCh)
	AssertClosed(t, ch, "after close")
}

func TestAssertionFailures(t *testing.T) {
	defer func(prev time.Duration) { negativeTimeout = prev }(negativeTimeout)
	negativeTimeout = 10 * time.Millisecond

	fn, ch := notifier.New()
	fail.AssertFails(t, func(t *testing.T) {
		AssertNotified(t, ch)
	}, "AssertNotified with no updates")

	fail.AssertFails(t, func(t *testing.T) {
		AssertClosed(t, ch)
	}, "AssertClosed with no updates")

	fn()
	fail.AssertFails(t, func(t *testing.T) {
		AssertClosed(t, ch)
	}, "AssertClosed with non-closing update")

	fn, ch = notifier.New()
	fn()

	fail.AssertFails(t, func(t *testing.T) {
		AssertNoUpdate(t, ch)
	}, "AssertNoUpdate with update")

	sendCh := make(chan struct{})
	ch = sendCh
	close(sendCh)

	fail.AssertFails(t, func(t *testing.T) {
		AssertNotified(t, ch)
	}, "AssertNotified after close")

	fail.AssertFails(t, func(t *testing.T) {
		AssertNoUpdate(t, ch)
	}, "AssertNoUpdate after close")
}
