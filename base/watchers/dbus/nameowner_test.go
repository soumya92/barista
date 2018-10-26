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

	"barista.run/testing/notifier"

	"github.com/godbus/dbus"
	"github.com/stretchr/testify/require"
)

// For testing, a service that acquires and releases names.
type testDBusNameService struct {
	*testing.T
	conn  *dbus.Conn
	names map[string]bool
}

func newTestDBusNameService(t *testing.T) *testDBusNameService {
	sessionBus, err := dbus.SessionBusPrivate()
	require.NoError(t, err, "dbus.SessionBusPrivate")
	require.NoError(t, sessionBus.Auth(nil), "sessionBus.Auth(nil)")
	require.NoError(t, sessionBus.Hello(), "sessionBus.Hello()")
	return &testDBusNameService{t, sessionBus, map[string]bool{}}
}

func (t *testDBusNameService) acquire(name string) {
	r, err := t.conn.RequestName(name, 0)
	require.NoError(t, err, "t.conn.RequestName")
	if r == dbus.RequestNameReplyPrimaryOwner {
		t.names[name] = true
	}
}

func (t *testDBusNameService) release(name string) {
	r, err := t.conn.ReleaseName(name)
	require.NoError(t, err, "t.conn.ReleaseName")
	if r == dbus.ReleaseNameReplyReleased {
		delete(t.names, name)
	}
}

func (t *testDBusNameService) clear() {
	for n := range t.names {
		r, err := t.conn.ReleaseName(n)
		require.NoError(t, err, "t.conn.ReleaseName")
		require.Equal(t, dbus.ReleaseNameReplyReleased, r, "ReleaseName(%s)", n)
	}
	t.names = map[string]bool{}
}

func TestSingleNameOwnerWatch(t *testing.T) {
	s := newTestDBusNameService(t)

	w, err := WatchNameOwner("org.i3barista.test.Service")
	defer w.Unsubscribe()
	require.NoError(t, err)

	notifier.AssertNoUpdate(t, w.C, "on start")
	require.Empty(t, w.GetOwner(), "no owner")

	s.acquire("org.i3barista.test.Service2")
	notifier.AssertNoUpdate(t, w.C, "different name acquired")
	require.Empty(t, w.GetOwner(), "still no owner")

	s.acquire("org.i3barista.test.Service")
	notifier.AssertNotified(t, w.C, "name acquired")
	require.NotEmpty(t, w.GetOwner(), "has owner")

	w2, err := WatchNameOwner("org.i3barista.test.Service")
	require.NoError(t, err)

	notifier.AssertNoUpdate(t, w2.C, "on start")
	require.NotEmpty(t, w2.GetOwner(), "has owner on start")

	w2.Unsubscribe()
	notifier.AssertNoUpdate(t, w2.C, "on unsubscribe")

	s.clear()
	notifier.AssertNotified(t, w.C, "name released")
	notifier.AssertNoUpdate(t, w2.C, "after unsubscribe")

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
	s := newTestDBusNameService(t)

	w, err := WatchNameOwners("org.i3barista")
	defer w.Unsubscribe()
	require.NoError(t, err)

	notifier.AssertNoUpdate(t, w.C, "on start")
	require.Empty(t, keys(w), "no owner")

	s.acquire("org.i3barista.test.Service")
	notifier.AssertNotified(t, w.C, "name acquired within namespace")
	require.Equal(t,
		[]string{"org.i3barista.test.Service"},
		keys(w))

	s.acquire("org.i3barista.test.Service2")
	notifier.AssertNotified(t, w.C, "another name acquired")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w))

	s.acquire("run.barista.test.Foo")
	notifier.AssertNoUpdate(t, w.C, "name acquired outside namespace")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w))

	w2, err := WatchNameOwners("org.i3barista.test")
	require.NoError(t, err)

	notifier.AssertNoUpdate(t, w2.C, "on start")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w2))

	w2.Unsubscribe()
	notifier.AssertNoUpdate(t, w2.C, "on unsubscribe")

	s.clear()
	notifier.AssertNotified(t, w.C, "name released")
	notifier.AssertNoUpdate(t, w2.C, "after unsubscribe")

	require.Empty(t, w.GetOwners(), "no owners")
}
