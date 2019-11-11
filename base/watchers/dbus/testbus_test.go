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
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

func assertSignalled(t *testing.T, ch <-chan *dbus.Signal, formatAndArgs ...interface{}) *dbus.Signal {
	select {
	case s := <-ch:
		drain(ch)
		return s
	case <-time.After(time.Second):
		require.Fail(t, "No signal received", formatAndArgs...)
	}
	return nil
}

func assertNotSignalled(t *testing.T, ch <-chan *dbus.Signal, formatAndArgs ...interface{}) {
	select {
	case <-ch:
		require.Fail(t, "Unexpected signal received", formatAndArgs...)
	case <-time.After(10 * time.Millisecond):
	}
}

func drain(ch <-chan *dbus.Signal) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}

func TestSignalMatching(t *testing.T) {
	b := SetupTestBus()

	s := b.RegisterService("org.i3barista.Service")
	obj := s.Object("/org/i3barista/Object", "")

	conn0 := Test()
	c := conn0.BusObject().AddMatchSignal("org.i3barista.Service", "Output",
		dbus.WithMatchOption("path_namespace", "/org/i3barista"),
		dbus.WithMatchOption("arg1", "foo"),
	)
	require.NoError(t, c.Err)

	conn1 := Test()
	c = conn1.BusObject().AddMatchSignal("org.i3barista.Service", "Output",
		dbus.WithMatchOption("path", "/org/i3barista/Object"),
		dbus.WithMatchOption("arg2namespace", "run.barista.sink"),
	)
	require.NoError(t, c.Err)

	conn2 := Test()
	c = conn2.BusObject().Call("GetNameOwner", 0, "org.i3barista.Service")
	require.NoError(t, c.Err)
	owner := c.Body[0].(string)

	c = conn2.BusObject().AddMatchSignal("org.i3barista.Service", "Output",
		dbus.WithMatchOption("arg0path", "/run/barista/sink"),
	)
	require.NoError(t, c.Err)

	c = conn2.BusObject().AddMatchSignal(
		"org.freedesktop.DBus.Properties", "PropertiesChanged",
		dbus.WithMatchOption("sender", owner),
	)
	require.NoError(t, c.Err)

	c = conn2.BusObject().AddMatchSignal(
		"org.freedesktop.DBus.Properties", "PropertiesChanged",
		dbus.WithMatchOption("path", "/org/i3barista"),
	)
	require.NoError(t, c.Err)

	c = Test().BusObject().AddMatchSignal("org.i3barista.Service", "Output",
		dbus.WithMatchOption("invalid", "argument"))
	require.Error(t, c.Err, "on invalid match option")

	c = conn2.BusObject().RemoveMatchSignal("org.i3barista.Service", "Signal")
	require.Error(t, c.Err, "removing non-existent match")

	sgn0 := make(chan *dbus.Signal, 10)
	sgn1 := make(chan *dbus.Signal, 10)
	sgn2 := make(chan *dbus.Signal, 10)

	conn0.Signal(sgn0)
	conn1.Signal(sgn1)
	conn2.Signal(sgn2)

	assertNotSignalled(t, sgn0, "on start")
	assertNotSignalled(t, sgn1)
	assertNotSignalled(t, sgn2)

	obj.Emit("Signal")
	assertNotSignalled(t, sgn0, "no match")
	assertNotSignalled(t, sgn1)
	assertNotSignalled(t, sgn2)

	obj.Emit("Output",
		dbus.ObjectPath("/run/barista/sink/System"),
		"foo",
		"run.barista.sink.Sink",
	)
	assertSignalled(t, sgn0, "arg1=")
	assertSignalled(t, sgn1, "arg2namespace")
	assertSignalled(t, sgn2, "arg0path")

	obj.SetPropertyForTest("anything", "value", SignalTypeChanged)
	assertSignalled(t, sgn2, "PropertiesChanged with sender filter")

	s2 := b.RegisterService("run.barista.sink.Sink")
	s2.Object("/run/barista/sink/System", "").
		SetPropertyForTest("foo", "baz", SignalTypeChanged)
	assertNotSignalled(t, sgn2, "PropertiesChanged from different sender")

	obj.SetPropertyForTest("anything", "othervalue", SignalTypeNone)
	assertNotSignalled(t, sgn2, "SetPropertyForTest without signal")

	objp := s.Object("/org/i3barista/Bar", "org.freedesktop.DBus.Properties")
	objp.Emit("PropertiesChanged", dbus.MakeVariant(map[string]interface{}{}))
	assertSignalled(t, sgn2, "PropertiesChanged sent manually")

	c = conn0.BusObject().AddMatchSignal("org.i3barista.Service", "Output",
		dbus.WithMatchOption("arg0", "4"))
	require.NoError(t, c.Err)
	obj.Emit("Output", 4)
	assertNotSignalled(t, sgn0, "int does not match string")
	assertNotSignalled(t, sgn1, "no match")
	assertNotSignalled(t, sgn2)

	conn2.RemoveSignal(sgn2)
	sgn2b := make(chan *dbus.Signal, 10)
	conn2.Signal(sgn2b)

	obj.SetPropertyForTest("anything", "newvalue", SignalTypeChanged)
	assertNotSignalled(t, sgn2, "PropertiesChanged after removing signal handler")
	assertSignalled(t, sgn2b, "PropertiesChanged on newly registered signal handler")
}
