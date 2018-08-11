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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValue(t *testing.T) {
	require := require.New(t)
	var v Value

	require.NotPanics(func() { v.Get() }, "Without a value set")
	require.Nil(v.Get(), "Unset value returns nil")

	v.Set("foobar")
	require.Equal("foobar", v.Get())

	require.Panics(func() { v.Set(int64(5)) },
		"Setting value of different type panics")
}

func TestValueUpdate(t *testing.T) {
	require := require.New(t)
	var v Value
	listening := make(chan bool)
	notified := make(chan bool)

	go func() {
		<-listening
		<-v.Update()
		notified <- true
	}()
	listening <- true

	v.Set("test")

	select {
	case <-notified:
	// Test passed, channel from v.Update was notified.
	case <-time.After(time.Second):
		require.Fail("<-Update() not notified within 1s")
	}

	select {
	case <-v.Update():
		require.Fail("<-Update() triggered without a Set(...)")
	case <-time.After(10 * time.Millisecond):
		// Test passed, Update() only notify of values
		// set after the call to Update.
	}

	v.Set("...")
	select {
	case <-v.Update():
		// Test passed, should notify since value was set.
	case <-time.After(time.Second):
		require.Fail("<-Update() notified of value")
	}
}

func TestErrorValue(t *testing.T) {
	require := require.New(t)
	var v ErrorValue

	require.NotPanics(func() { v.Get() }, "Without a value/error set")
	val, err := v.Get()
	require.Nil(val, "Empty state returns nil value")
	require.NoError(err, "Empty state returns nil error")

	v.Set("foobar")
	val, err = v.Get()
	require.Equal("foobar", val)
	require.NoError(err, "When value was set")

	// TODO: This should work.
	// require.Panics(func() { v.Set(int64(5)) },
	// 	"Setting value of different type panics")

	require.True(v.Error(fmt.Errorf("blah")),
		"Error returns true for non-nil error")
	val, err = v.Get()
	require.Nil(val, "Error returns nil value")
	require.Error(err)

	v.Set("...")
	val, err = v.Get()
	require.NoError(err, "Setting value clears error")

	require.False(v.Error(nil), "Error returns false for nil error")
	val, err = v.Get()
	require.NoError(err, "After Error(nil)")
	require.Equal("...", val, "Value unchanged after Error(nil)")
}

func TestErrorValueSubscription(t *testing.T) {
	require := require.New(t)
	var v ErrorValue

	subChan := make(chan error)
	go func() {
		for range v.Update() {
			_, err := v.Get()
			subChan <- err
		}
	}()

	select {
	case <-subChan:
		require.Fail("Received update with no value set")
	case <-time.After(10 * time.Millisecond):
		// Test passed.
	}

	v.Set("Test")
	select {
	case err := <-subChan:
		require.NoError(err, "On value set")
	case <-time.After(time.Second):
		require.Fail("<-Update() not notified within 1s")
	}

	v.Error(nil)
	select {
	case <-subChan:
		require.Fail("Received update after Error(nil)")
	case <-time.After(10 * time.Millisecond):
		// Test passed, Error(nil) does not change the value,
		// so should not notify.
	}

	v.Error(fmt.Errorf("xx"))
	select {
	case err := <-subChan:
		require.Error(err, "On Error(non-nil)")
	case <-time.After(time.Second):
		require.Fail("<-Update() not notified within 1s")
	}
}
