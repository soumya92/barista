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
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

func TestServices(t *testing.T) {
	b := SetupTestBus()

	conn0 := Test()
	sgn0 := make(chan *dbus.Signal, 10)
	c := nameOwnerChanged.addMatch(conn0,
		dbus.WithMatchOption("arg0", "org.i3barista.Service"))
	require.NoError(t, c.Err)
	conn0.Signal(sgn0)

	conn1 := Test()
	sgn1 := make(chan *dbus.Signal, 10)
	c = nameOwnerChanged.addMatch(conn1,
		dbus.WithMatchOption("arg0namespace", "org.i3barista.Misc"))
	require.NoError(t, c.Err)
	conn1.Signal(sgn1)

	assertNotSignalled(t, sgn0, "on start")
	assertNotSignalled(t, sgn1)

	svc0 := b.RegisterService()
	assertNotSignalled(t, sgn0, "unnamed service registered")
	assertNotSignalled(t, sgn1)

	svc1 := b.RegisterService("org.i3barista.Misc.FooService")
	assertNotSignalled(t, sgn0, "Name does not match")
	assertSignalled(t, sgn1, "Within expected namespace")

	svc0.AddName("org.i3barista.Misc.BarService")
	assertNotSignalled(t, sgn0, "Name does not match")
	assertSignalled(t, sgn1, "Within expected namespace")

	svc0.AddName("org.i3barista.Service")
	assertSignalled(t, sgn0, "Name matches")
	assertNotSignalled(t, sgn1, "Outside namespace")

	svc0.Unregister()
	assertSignalled(t, sgn0, "Matching service unregistered")
	assertSignalled(t, sgn1, "Service within namespace unregistered")

	c = nameOwnerChanged.addMatch(conn0,
		dbus.WithMatchOption("arg0namespace", "org.i3barista.Misc"))
	require.NoError(t, c.Err)
	svc1.AddName("org.i3barista.Misc.FooServiceAlias")

	assertSignalled(t, sgn0, "Within expected namespace")
	assertSignalled(t, sgn1, "Within expected namespace")

	c = nameOwnerChanged.removeMatch(conn1,
		dbus.WithMatchOption("arg0namespace", "org.i3barista.Misc"))
	require.NoError(t, c.Err)
	svc1.RemoveName("org.i3barista.Misc.FooService")

	assertSignalled(t, sgn0, "Within expected namespace")
	assertNotSignalled(t, sgn1, "All matches removed")

	svc2 := b.RegisterService("org.i3barista.Misc.FooServiceAlias")
	s := assertSignalled(t, sgn0, "Within expected namespace")
	require.NotEmpty(t, s.Body[1], "has previous owner")
	require.NotEmpty(t, s.Body[2], "has new owner")
	assertNotSignalled(t, sgn1, "All matches removed")

	require.NoError(t, conn0.Close())
	require.NoError(t, conn1.Close())
	require.Error(t, conn1.Close(), "closing already-closed connection")

	assertNotSignalled(t, sgn0, "On close")
	assertNotSignalled(t, sgn1)

	svc1.AddName("org.i3barista.Misc.BazService")
	svc2.AddName("org.i3barista.Misc.BazService")

	c = getNameOwner.call(Test(), "org.i3barista.Misc.BazService")
	require.NoError(t, c.Err)
	require.Equal(t, svc2.id, c.Body[0])
	require.Empty(t, svc1.names)

	c = getNameOwner.call(Test(), "run.barista.NoSuchService")
	require.Error(t, c.Err)
	require.Empty(t, c.Body)

	require.NotPanics(t, func() {
		svc1.RemoveName("org.i3barista.Misc.BazService")
	}, "removing non-existent name")
	require.NotPanics(t, func() {
		svc2.AddName("org.i3barista.Misc.BazService")
	}, "re-adding owned name")
	require.Panics(t, func() {
		getNameOwner.call(conn0, "run.barista.NoSuchService")
	}, "attempting to call method on closed connection")

	obj := svc1.Object("/org/i3barista/test/Foo", "")

	svc1.Unregister()
	svc2.Unregister()

	assertNotSignalled(t, sgn0, "After close")
	assertNotSignalled(t, sgn1)

	require.Panics(t, func() {
		svc1.Object("/org/i3barista/test/Object", "")
	}, "use after unregister")

	require.Panics(t, func() {
		obj.SetPropertyForTest("foo", "baz", SignalTypeNone)
	}, "use object after unregister")

	require.Panics(t, func() {
		svc2.Unregister()
	}, "duplicate unregister")
}
