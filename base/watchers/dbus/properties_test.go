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

	"github.com/godbus/dbus"
	"github.com/stretchr/testify/require"
)

func assertUpdated(t *testing.T, w *PropertiesWatcher, formatAndArgs ...interface{}) PropertiesChange {
	select {
	case c := <-w.Updates:
		return c
	case <-time.After(time.Second):
		require.Fail(t, "PropertiesWatcher not updated", formatAndArgs...)
	}
	return nil
}

func assertNotUpdated(t *testing.T, w *PropertiesWatcher, formatAndArgs ...interface{}) {
	select {
	case <-w.Updates:
		require.Fail(t, "PropertiesWatcher unexpectedly updated", formatAndArgs...)
	case <-time.After(10 * time.Millisecond):
	}
}

func TestProperties(t *testing.T) {
	bus := SetupTestBus()
	srv := bus.RegisterService("org.i3barista.services.FooService")

	conn := Test() // Needed to ensure all signals are drained.
	nameOwnerChanged.addMatch(conn,
		dbus.WithMatchOption("arg0", "org.i3barista.services.FooService"))
	ch := make(chan *dbus.Signal, 10)
	conn.Signal(ch)

	obj := srv.Object("/org/i3barista/objects/Foo", "org.i3barista.Service")
	obj.SetProperty("a", 1, false)
	<-ch // NameOwnerChanged.

	w := WatchProperties(Test,
		"org.i3barista.services.FooService",
		"/org/i3barista/objects/Foo",
		"org.i3barista.Service",
		[]string{"a", "b", "c", "d", "fromSignal", "fetched"},
	)

	assertNotUpdated(t, w, "on start")
	require.Equal(t, map[string]interface{}{
		"a": 1,
	}, w.Get(), "Initial values")

	obj.SetProperty("a", 2, true)
	u := assertUpdated(t, w, "On property change with signal")
	require.Equal(t, PropertiesChange{"a": {1, 2}}, u,
		"Old value and new value in update")

	obj.SetProperty("d", "baz", false)
	assertNotUpdated(t, w, "On property change without signal")

	obj.SetProperty("c", "anotherstring", true)
	u = assertUpdated(t, w, "On property change with signal")
	require.Equal(t, PropertiesChange{"c": {nil, "anotherstring"}}, u,
		"Nil old value for newly set property")

	require.Equal(t, map[string]interface{}{
		"a": 2,
		"c": "anotherstring",
	}, w.Get(), "Non-signal property change ignored")

	obj.SetProperty("foo", "whatever", true)
	assertNotUpdated(t, w, "On uninteresting property change")

	obj.SetProperty("d", 5, true)
	u = assertUpdated(t, w, "On property change with signal")
	require.Equal(t, PropertiesChange{"d": {nil, 5}}, u,
		"Nil old value for previously ignored set")

	srv1 := bus.RegisterService()
	obj = srv1.Object("/org/i3barista/objects/Foo", "org.i3barista.Service")
	obj.SetProperty("a", 4, false)
	obj.SetProperty("b", "banana", false)
	obj.SetProperty("d", 5, true)

	srv1.AddName("org.i3barista.services.FooService") // Replace previous.
	u = assertUpdated(t, w, "On service move")
	require.Equal(t, PropertiesChange{
		"a": {2, 4},
		"b": {nil, "banana"},
		"c": {"anotherstring", nil},
		"d": {5, 5},
	}, u, "Values compared between objects")

	obj.On("Method", func(args ...interface{}) ([]interface{}, error) {
		return []interface{}{1, "2", args[0]}, nil
	})

	obj.Emit("Signal", 4)
	assertNotUpdated(t, w, "On unhandled signal")

	r, err := w.Call("Method", "foo")
	require.NoError(t, err, "On method call")
	require.Equal(t, []interface{}{1, "2", "foo"}, r)

	_, err = w.Call("Undefined", 3, 1, 4)
	require.Error(t, err, "On undefined method call")

	obj.SetProperty("d", 7, true)
	u = assertUpdated(t, w, "On signal after move")
	require.Equal(t, PropertiesChange{"d": {5, 7}}, u)

	w.AddSignalHandler("Signal",
		func(s *Signal, f Fetcher) map[string]interface{} {
			val, _ := f("foo")
			return map[string]interface{}{
				"fromSignal": s.Body[0],
				"fetched":    val,
			}
		})

	obj.Emit("Signal", 5)
	u = assertUpdated(t, w, "On signal handler")
	require.Equal(t, PropertiesChange{
		"fromSignal": {nil, 5},
		"fetched":    {nil, nil},
	}, u)

	obj.SetProperty("foo", 0, false)
	obj.Emit("Signal", 8)
	u = assertUpdated(t, w, "On signal handler")
	require.Equal(t, PropertiesChange{
		"fromSignal": {5, 8},
		"fetched":    {nil, 0},
	}, u)

	srv1.RemoveName("org.i3barista.services.FooService")
	u = assertUpdated(t, w, "On service disconnect")
	require.Equal(t, PropertiesChange{
		"a":          {4, nil},
		"b":          {"banana", nil},
		"d":          {7, nil},
		"fromSignal": {8, nil},
		"fetched":    {0, nil},
	}, u, "All properties cleared")
	require.Empty(t, w.Get())

	obj.Emit("Signal", "c")
	assertNotUpdated(t, w, "After service disconnect")

	_, err = w.Call("Method", 3, 1, 4)
	require.Error(t, err, "On method call while disconnected")

	srv.AddName("org.i3barista.services.FooService")
	assertUpdated(t, w, "On service connect")
	require.Equal(t, map[string]interface{}{
		"a": 2,
		"c": "anotherstring",
		"d": 5,
	}, w.Get())

	w.Unsubscribe()
	require.Empty(t, w.Get(), "After Unsubscribe")

	obj.SetProperty("a", 4, true)
	assertNotUpdated(t, w, "after Unsubscribe")

	obj.Emit("Signal", "foo")
	assertNotUpdated(t, w, "after Unsubscribe")
}
