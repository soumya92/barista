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

package base

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/notifier"
)

func TestSubscription(t *testing.T) {
	require := require.New(t)
	notifyFn, notifyCh := notifier.New()
	e := Multicast(notifyCh)

	var listening sync.WaitGroup
	var notified sync.WaitGroup

	for i := 0; i < 25; i++ {
		subI := e.Subscribe()
		listening.Add(1)
		go func() {
			listening.Done()
			<-subI
			notified.Done()
		}()
		notified.Add(1)
	}
	listening.Wait()
	doneChan := make(chan bool)
	go func() {
		notified.Wait()
		doneChan <- true
	}()

	notifyFn()

	select {
	case <-doneChan:
	// Test passed, all 25 subscriptions were notified.
	case <-time.After(time.Second):
		require.Fail("Subscriptions not notified within 1s")
	}

	newSub := e.Subscribe()
	select {
	case <-newSub:
		require.Fail("Newly created subscription notified")
	case <-time.After(10 * time.Millisecond):
		// Test passed, subscriptions only notify of values
		// set after the call to Subscribe.
	}

	notifyFn()
	select {
	case <-newSub:
		// Test passed, should notify since value was set.
	case <-time.After(time.Second):
		require.Fail("New subscription was not notified of value")
	}
}

func TestNext(t *testing.T) {
	require := require.New(t)
	notifyFn, notifyCh := notifier.New()
	e := Multicast(notifyCh)

	var listening sync.WaitGroup
	var notified sync.WaitGroup

	for i := 0; i < 25; i++ {
		listening.Add(1)
		go func() {
			listening.Done()
			<-e.Next()
			notified.Done()
		}()
		notified.Add(1)
	}
	listening.Wait()
	doneChan := make(chan bool)
	go func() {
		notified.Wait()
		doneChan <- true
	}()

	notifyFn()

	select {
	case <-doneChan:
	// Test passed, all 25 subscriptions were notified.
	case <-time.After(time.Second):
		require.Fail("Subscriptions not notified within 1s")
	}

	select {
	case <-e.Next():
		require.Fail("Next() notified without changes")
	case <-time.After(10 * time.Millisecond):
	}

	notifyFn()
	<-e.Subscribe()

	select {
	case <-e.Next():
		require.Fail("Next notified for stale value")
	case <-time.After(10 * time.Millisecond):
	}

	next := e.Next()
	notifyFn()
	select {
	case <-next:
		// Should notify, because call to Next was before notification.
	case <-time.After(10 * time.Millisecond):
		require.Fail("Next not notified on subsequent value")
	}
}
