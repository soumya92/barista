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

	"barista.run/testing/notifier"
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

func TestSignal(t *testing.T) {
	s := new(Signaller)
	beforeSignal0 := s.Next()
	beforeSignal1 := s.Next()
	s.Signal()
	afterSignal := s.Next()

	notifier.AssertClosed(t, beforeSignal0, "Next() before Signal()")
	notifier.AssertClosed(t, beforeSignal1, "Next() before Signal()")
	notifier.AssertNoUpdate(t, afterSignal, "Next() after Signal()")
}
