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

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/notifier"
)

func TestSubscription(t *testing.T) {
	assert := assert.New(t)
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
		assert.Fail("Subscriptions not notified within 1s")
	}

	newSub := e.Subscribe()
	select {
	case <-newSub:
		assert.Fail("Newly created subscription notified")
	case <-time.After(10 * time.Millisecond):
		// Test passed, subscriptions only notify of values
		// set after the call to Subscribe.
	}

	notifyFn()
	select {
	case <-newSub:
		// Test passed, should notify since value was set.
	case <-time.After(time.Second):
		assert.Fail("New subscription was not notified of value")
	}
}
