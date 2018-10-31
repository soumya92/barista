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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/godbus/dbus"
	"github.com/stretchr/testify/require"
)

func TestObjects(t *testing.T) {
	b := SetupTestBus()

	svc0 := b.RegisterService("org.i3barista.Misc.BarService")
	o0 := svc0.Object("/org/i3barista/Misc/Bar", "")
	require.Equal(t, o0.Destination(), "org.i3barista.Misc.BarService")

	svc1 := b.RegisterService("org.i3barista.Misc.FooService")
	o1 := svc1.Object("/org/i3barista/Misc/Foo", "")
	require.Equal(t, o1.Destination(), "org.i3barista.Misc.FooService")

	svc2 := b.RegisterService()
	o2 := svc2.Object("/org/i3barista/Bar", "")
	require.Empty(t, o2.Destination(), "empty destination for unnamed service")
	require.Equal(t, dbus.ObjectPath("/org/i3barista/Bar"), o2.Path())

	o2p := svc2.Object("/org/i3barista/Bar2", "org.freedesktop.DBus.Properties")
	require.Equal(t, "org.freedesktop.DBus.Properties", o2p.Destination(),
		"overridden destination service name")

	o0.SetProperty("color", "red", false)
	val, err := Test().
		Object("org.i3barista.Misc.BarService", "/org/i3barista/Misc/Bar").
		GetProperty("org.i3barista.Misc.BarService.color")
	require.NoError(t, err)
	require.Equal(t, dbus.MakeVariant("red"), val)

	_, err = Test().
		Object("org.i3barista.Misc.BarService", "/org/i3barista/Misc/Bar").
		GetProperty("org.i3barista.Misc.BarService.nosuchproperty")
	require.Error(t, err)

	require.Panics(t, func() {
		Test().Object("run.barista.NoSuchService", "/run/barista/Foo")
	}, "Unknown service")

	conn := Test()
	c := conn.
		Object("org.i3barista.Misc.BarService", "/org/i3barista/Misc/Bar").
		Call("Method", 0, "arg0", 1, 2.1)
	require.Error(t, c.Err, "method not defined")

	o0.On("Method", func(args ...interface{}) ([]interface{}, error) {
		if len(args) != 2 {
			return nil, errors.New("something")
		}
		require.Equal(t, "arg0", args[0])
		require.Equal(t, 1, args[1])
		return []interface{}{"0", 1, 2.0}, nil
	})
	c = conn.
		Object("org.i3barista.Misc.BarService", "/org/i3barista/Misc/Bar").
		CallWithContext(context.TODO(), "Method", 0, "arg0", 1, 2.1)
	require.Error(t, c.Err, "method returned error")

	ch := make(chan *dbus.Call, 10)
	conn.
		Object("org.i3barista.Misc.BarService", "/org/i3barista/Misc/Bar").
		GoWithContext(context.TODO(), "Method", 0, ch, "arg0", 1)

	select {
	case <-ch:
		require.Fail(t, "Unexpected value on Go() channel")
	case <-time.After(10 * time.Millisecond): // expected.
	}

	select {
	case c = <-ch:
	case <-time.After(time.Second):
		require.Fail(t, "No value received on Go() channel")
	}
	var str string
	var num int
	var dbl float64
	require.NoError(t, c.Store(&str, &num, &dbl))
	require.Equal(t, "0", str)
	require.Equal(t, 1, num)
	require.Equal(t, 2.0, dbl)
}
