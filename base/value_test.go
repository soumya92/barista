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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"
)

func TestValue(t *testing.T) {
	assert := assert.New(t)
	var v Value

	assert.NotPanics(func() { v.Get() }, "Without a value set")
	assert.Nil(v.Get(), "Unset value returns nil")

	v.Set("foobar")
	assert.Equal("foobar", v.Get())

	assert.Panics(func() { v.Set(int64(5)) },
		"Setting value of different type panics")
}

func TestValueSubscription(t *testing.T) {
	assert := assert.New(t)
	var v Value
	var listening sync.WaitGroup
	var notified sync.WaitGroup

	for i := 0; i < 25; i++ {
		subI := v.Subscribe()
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
		listening.Wait()
		doneChan <- true
	}()

	v.Set("test")

	select {
	case <-doneChan:
	// Test passed, all 25 subscriptions were notified.
	case <-time.After(time.Second):
		assert.Fail("Subscriptions not notified within 1s")
	}

	newSub := v.Subscribe()
	select {
	case <-newSub:
		assert.Fail("Newly created subscription notified")
	case <-time.After(10 * time.Millisecond):
		// Test passed, subscriptions only notify of values
		// set after the call to Subscribe.
	}

	v.Set("...")
	select {
	case <-newSub:
		// Test passed, should notify since value was set.
	case <-time.After(time.Second):
		assert.Fail("New subscription was not notified of value")
	}
}

func TestErrorValue(t *testing.T) {
	assert := assert.New(t)
	var v ErrorValue

	assert.NotPanics(func() { v.Get() }, "Without a value/error set")
	val, err := v.Get()
	assert.Nil(val, "Empty state returns nil value")
	assert.NoError(err, "Empty state returns nil error")

	v.Set("foobar")
	val, err = v.Get()
	assert.Equal("foobar", val)
	assert.NoError(err, "When value was set")

	// TODO: This should work.
	// assert.Panics(func() { v.Set(int64(5)) },
	// 	"Setting value of different type panics")

	assert.True(v.Error(fmt.Errorf("blah")),
		"Error returns true for non-nil error")
	val, err = v.Get()
	assert.Nil(val, "Error returns nil value")
	assert.Error(err)

	v.Set("...")
	val, err = v.Get()
	assert.NoError(err, "Setting value clears error")

	assert.False(v.Error(nil), "Error returns false for nil error")
	val, err = v.Get()
	assert.NoError(err, "After Error(nil)")
	assert.Equal("...", val, "Value unchanged after Error(nil)")
}

func TestErrorValueSubscription(t *testing.T) {
	assert := assert.New(t)
	var v ErrorValue

	subChan := make(chan error)
	sub := v.Subscribe()
	go func() {
		for range sub {
			_, err := v.Get()
			subChan <- err
		}
	}()

	select {
	case <-subChan:
		assert.Fail("Received subscription with no value set")
	case <-time.After(10 * time.Millisecond):
		// Test passed.
	}

	v.Set("Test")
	select {
	case err := <-subChan:
		assert.NoError(err, "On value set")
	case <-time.After(time.Second):
		assert.Fail("Subscription not notified within 1s")
	}

	v.Error(nil)
	select {
	case <-subChan:
		assert.Fail("Received subscription after Error(nil)")
	case <-time.After(10 * time.Millisecond):
		// Test passed, Error(nil) does not change the value,
		// so should not notify.
	}

	v.Error(fmt.Errorf("xx"))
	select {
	case err := <-subChan:
		assert.Error(err, "On Error(non-nil)")
	case <-time.After(time.Second):
		assert.Fail("Subscription not notified within 1s")
	}
}
