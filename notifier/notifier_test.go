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

	"github.com/soumya92/barista/bar"
	"github.com/stretchrcom/testify/assert"
)

func assertTick(t *testing.T, n bar.Notifier, message string) {
	select {
	case <-n.Tick():
	case <-time.After(time.Second):
		assert.Fail(t, "notifier did not update", message)
	}
}

func assertNoTick(t *testing.T, n bar.Notifier, message string) {
	select {
	case <-n.Tick():
		assert.Fail(t, "notifier updated", message)
	case <-time.After(10 * time.Millisecond):
	}
}

func TestSimpleNotify(t *testing.T) {
	n := New()
	n.Notify()
	assertTick(t, n, "when notified")
	assertNoTick(t, n, "when not notified")
}

func TestMultipleNotify(t *testing.T) {
	n := New()
	for i := 0; i < 5; i++ {
		n.Notify()
	}
	assertTick(t, n, "when notified")
	assertNoTick(t, n, "multiple notifications are merged")
}

func TestNotifyWithWaiting(t *testing.T) {
	n := New()

	var launched sync.WaitGroup
	var waited sync.WaitGroup
	for i := 0; i < 5; i++ {
		launched.Add(1)
		waited.Add(1)
		go func() {
			launched.Done()
			n.Wait()
			waited.Done()
		}()
	}
	launched.Wait()
	for i := 0; i < 5; i++ {
		n.Notify()
	}
	doneChan := make(chan interface{})
	go func() {
		waited.Wait()
		doneChan <- nil
	}()

	select {
	case <-doneChan: // Test passed.
	case <-time.After(time.Second):
		assert.Fail(t, "waits did not complete")
	}
}

func TestWait(t *testing.T) {
	n := New()
	n.Notify()

	// Notify was already called,
	// Wait() should return immediately.
	doneChan := make(chan interface{})
	go func() {
		n.Wait()
		doneChan <- nil
	}()

	select {
	case <-doneChan: // Test passed.
	case <-time.After(time.Second):
		assert.Fail(t, "wait did not complete")
	}
}
