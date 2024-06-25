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
	"sync"
	"testing"
	"time"

	"github.com/soumya92/barista/testing/notifier"
	"github.com/stretchr/testify/require"
)

func TestSimpleNotify(t *testing.T) {
	fn, n := New()
	fn()
	notifier.AssertNotified(t, n, "when notified")
	notifier.AssertNoUpdate(t, n, "when not notified")
}

func TestMultipleNotify(t *testing.T) {
	fn, n := New()
	for i := 0; i < 5; i++ {
		fn()
	}
	notifier.AssertNotified(t, n, "when notified")
	notifier.AssertNoUpdate(t, n, "multiple notifications are merged")
}

func TestNotifyWithWaiting(t *testing.T) {
	fn, n := New()

	var launched sync.WaitGroup
	var waited sync.WaitGroup
	for i := 0; i < 5; i++ {
		launched.Add(1)
		waited.Add(1)
		go func() {
			launched.Done()
			<-n
			waited.Done()
		}()
	}
	launched.Wait()
	for i := 0; i < 5; i++ {
		fn()
	}
	doneChan := make(chan struct{})
	go func() {
		waited.Wait()
		doneChan <- struct{}{}
	}()

	select {
	case <-doneChan: // Test passed.
	case <-time.After(time.Second):
		require.Fail(t, "waits did not complete")
	}
}

func TestWait(t *testing.T) {
	fn, n := New()
	fn()

	// Already notified, <- should return immediately.
	doneChan := make(chan struct{})
	go func() {
		<-n
		doneChan <- struct{}{}
	}()

	select {
	case <-doneChan: // Test passed.
	case <-time.After(time.Second):
		require.Fail(t, "wait did not complete")
	}
}

func TestSourceNext(t *testing.T) {
	s := new(Source)
	beforeSignal0 := s.Next()
	beforeSignal1 := s.Next()
	s.Notify()
	afterSignal := s.Next()

	notifier.AssertClosed(t, beforeSignal0, "Next() before Notify()")
	notifier.AssertClosed(t, beforeSignal1, "Next() before Notify()")
	notifier.AssertNoUpdate(t, afterSignal, "Next() after Notify()")
}

func TestSourceSubscribe(t *testing.T) {
	s := new(Source)

	before, beforeDone := s.Subscribe()
	s.Notify()
	after, afterDone := s.Subscribe()

	notifier.AssertNotified(t, before, "Subscribe() before notification")
	notifier.AssertNoUpdate(t, after, "Subscribe() after notification")

	s.Notify()

	notifier.AssertNotified(t, before,
		"Previously notified subscription after another notification")
	notifier.AssertNotified(t, after, "Subscription after notification")

	beforeDone()
	s.Notify()

	notifier.AssertNoUpdate(t, before, "Subscription after done()")
	notifier.AssertNotified(t, after, "Different subscription still notified")

	require.NotPanics(t, func() { beforeDone() }, "Duplicate cleanup")

	afterDone()
	notifier.AssertNoUpdate(t, after, "On calling done()")
}
