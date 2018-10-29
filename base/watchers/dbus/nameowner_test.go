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

	"github.com/stretchr/testify/require"
)

func TestSingleNameOwnerWatch(t *testing.T) {
	s := SetupTestBus().RegisterService()

	w := WatchNameOwner(Test, "org.i3barista.test.Service")
	defer w.Unsubscribe()

	notifier.AssertNoUpdate(t, w.C, "on start")
	require.Empty(t, w.GetOwner(), "no owner")

	s.AddName("org.i3barista.test.Service2")
	notifier.AssertNoUpdate(t, w.C, "different name acquired")
	require.Empty(t, w.GetOwner(), "still no owner")

	s.AddName("org.i3barista.test.Service")
	notifier.AssertNotified(t, w.C, "name acquired")
	require.NotEmpty(t, w.GetOwner(), "has owner")

	w2 := WatchNameOwner(Test, "org.i3barista.test.Service")

	notifier.AssertNoUpdate(t, w2.C, "on start")
	require.NotEmpty(t, w2.GetOwner(), "has owner on start")

	w2.Unsubscribe()
	notifier.AssertNoUpdate(t, w2.C, "on unsubscribe")

	s.Unregister()
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
	s := SetupTestBus().RegisterService()

	w := WatchNameOwners(Test, "org.i3barista")
	defer w.Unsubscribe()

	notifier.AssertNoUpdate(t, w.C, "on start")
	require.Empty(t, keys(w), "no owner")

	s.AddName("org.i3barista.test.Service")
	notifier.AssertNotified(t, w.C, "name acquired within namespace")
	require.Equal(t,
		[]string{"org.i3barista.test.Service"},
		keys(w))

	s.AddName("org.i3barista.test.Service2")
	notifier.AssertNotified(t, w.C, "another name acquired")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w))

	s.AddName("run.barista.test.Foo")
	notifier.AssertNoUpdate(t, w.C, "name acquired outside namespace")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w))

	w2 := WatchNameOwners(Test, "org.i3barista.test")

	notifier.AssertNoUpdate(t, w2.C, "on start")
	require.Equal(t,
		[]string{"org.i3barista.test.Service", "org.i3barista.test.Service2"},
		keys(w2))

	w2.Unsubscribe()
	notifier.AssertNoUpdate(t, w2.C, "on unsubscribe")

	s.RemoveName("org.i3barista.test.Service")
	notifier.AssertNotified(t, w.C, "name released")
	notifier.AssertNoUpdate(t, w2.C, "after unsubscribe")

	s.RemoveName("org.i3barista.test.Service2")
	notifier.AssertNotified(t, w.C, "name released")
	require.Empty(t, w.GetOwners(), "no owners")
}
