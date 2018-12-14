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

package dbus

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func assertNotified(t *testing.T, ch <-chan NameOwnerChange, formatAndArgs ...interface{}) NameOwnerChange {
	select {
	case <-time.After(10 * time.Second):
		require.Fail(t, "Expected an update", formatAndArgs...)
	case u := <-ch:
		return u
	}
	return NameOwnerChange{}
}

func assertNoUpdate(t *testing.T, ch <-chan NameOwnerChange, formatAndArgs ...interface{}) {
	select {
	case <-time.After(10 * time.Millisecond):
		// test passed
	case <-ch:
		require.Fail(t, "Expected no update", formatAndArgs...)
	}
}

func TestSingleNameOwnerWatch(t *testing.T) {
	bus := SetupTestBus()
	s := bus.RegisterService()

	w := WatchNameOwner(Test, "org.i3barista.test.Service")
	defer w.Unsubscribe()

	assertNoUpdate(t, w.Updates, "on start")
	require.Empty(t, w.GetOwner(), "no owner")

	s.AddName("org.i3barista.test.Service2")
	assertNoUpdate(t, w.Updates, "different name acquired")
	require.Empty(t, w.GetOwner(), "still no owner")

	s.AddName("org.i3barista.test.Service")
	u := assertNotified(t, w.Updates, "name acquired")
	require.Equal(t, u.Name, "org.i3barista.test.Service")
	require.NotEmpty(t, u.Owner)
	require.NotEmpty(t, w.GetOwner(), "has owner")

	w2 := WatchNameOwner(Test, "org.i3barista.test.Service")

	assertNoUpdate(t, w2.Updates, "on start")
	require.NotEmpty(t, w2.GetOwner(), "has owner on start")

	// Need to make sure the listen() goroutine has started.
	s1 := bus.RegisterService("org.i3barista.test.Service")
	oldOwner := u.Owner
	u = assertNotified(t, w.Updates, "new owner")
	require.Equal(t, "org.i3barista.test.Service", u.Name)
	require.NotEqual(t, oldOwner, u.Owner)
	assertNotified(t, w2.Updates, "new owner")
	require.Equal(t, "org.i3barista.test.Service", u.Name)
	require.NotEqual(t, oldOwner, u.Owner)

	w2.Unsubscribe()
	assertNoUpdate(t, w2.Updates, "on unsubscribe")

	s1.Unregister()
	u = assertNotified(t, w.Updates, "name released")
	require.Equal(t, u.Name, "org.i3barista.test.Service")
	require.Empty(t, u.Owner)
	assertNoUpdate(t, w2.Updates, "after unsubscribe")

	require.Empty(t, w.GetOwner(), "no owner")
}

func keys(n *NameOwnerWatcher) []string {
	ks := []string{}
	for k := range n.GetOwners() {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func TestNamespacedOwnerWatch(t *testing.T) {
	s := SetupTestBus().RegisterService()

	w := WatchNameOwners(Test, "org.i3barista")
	defer w.Unsubscribe()

	assertNoUpdate(t, w.Updates, "on start")
	require.Empty(t, keys(w), "no owner")

	s.AddName("org.i3barista.test.Service")
	u := assertNotified(t, w.Updates, "name acquired within namespace")
	require.Equal(t,
		[]string{"org.i3barista.test.Service"},
		keys(w))
	require.Equal(t, "org.i3barista.test.Service", u.Name)
	require.NotEmpty(t, u.Owner)

	s.AddName("org.i3barista.test.Service2")
	u = assertNotified(t, w.Updates, "another name acquired")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w))
	require.Equal(t, "org.i3barista.test.Service2", u.Name)
	require.NotEmpty(t, u.Owner)

	s.AddName("run.barista.test.Foo")
	assertNoUpdate(t, w.Updates, "name acquired outside namespace")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w))

	w2 := WatchNameOwners(Test, "org.i3barista.test")

	assertNoUpdate(t, w2.Updates, "on start")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w2))

	w2.Unsubscribe()
	assertNoUpdate(t, w2.Updates, "on unsubscribe")

	s.RemoveName("org.i3barista.test.Service")
	u = assertNotified(t, w.Updates, "name released")
	require.Equal(t, "org.i3barista.test.Service", u.Name)
	require.Empty(t, u.Owner)
	assertNoUpdate(t, w2.Updates, "after unsubscribe")

	s.RemoveName("org.i3barista.test.Service2")
	u = assertNotified(t, w.Updates, "name released")
	require.Equal(t, "org.i3barista.test.Service2", u.Name)
	require.Empty(t, u.Owner)
	require.Empty(t, w.GetOwners(), "no owners")
}
