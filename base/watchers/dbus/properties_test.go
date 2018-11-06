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

	conn := Test() // Needed to ensure all signals are drained.
	nameOwnerChanged.addMatch(conn,
		dbus.WithMatchOption("arg0", "org.i3barista.services.FooService"))
	ch := make(chan *dbus.Signal, 10)
	conn.Signal(ch)

	srv := bus.RegisterService("org.i3barista.services.FooService")
	obj := srv.Object("/org/i3barista/objects/Foo", "org.i3barista.Service")
	obj.SetProperty("a", 1, SignalTypeNone)
	<-ch // NameOwnerChanged.

	w := WatchProperties(Test,
		"org.i3barista.services.FooService",
		"/org/i3barista/objects/Foo",
		"org.i3barista.Service").
		Add("a", "b", "c", "d", "fromSignal").
		Fetch("fetched")

	assertNotUpdated(t, w, "on start")
	require.Equal(t, map[string]interface{}{
		"a": 1,
	}, w.Get(), "Initial values")

	obj.SetProperty("a", 2, SignalTypeChanged)
	u := assertUpdated(t, w, "On property change with signal")
	require.Equal(t, PropertiesChange{"a": {1, 2}}, u,
		"Old value and new value in update")

	obj.SetProperty("d", "baz", SignalTypeNone)
	assertNotUpdated(t, w, "On property change without signal")

	obj.SetProperty("c", "anotherstring", SignalTypeInvalidated)
	u = assertUpdated(t, w, "On property change with signal")
	require.Equal(t, PropertiesChange{"c": {nil, "anotherstring"}}, u,
		"Nil old value for newly set property")

	require.Equal(t, map[string]interface{}{
		"a": 2,
		"c": "anotherstring",
	}, w.Get(), "Non-signal property change ignored")

	obj.SetProperty("foo", "whatever", SignalTypeInvalidated)
	assertNotUpdated(t, w, "On uninteresting property change")

	obj.SetProperty("d", 5, SignalTypeChanged)
	u = assertUpdated(t, w, "On property change with signal")
	require.Equal(t, PropertiesChange{"d": {nil, 5}}, u,
		"Nil old value for previously ignored set")

	srv1 := bus.RegisterService()
	obj = srv1.Object("/org/i3barista/objects/Foo", "org.i3barista.Service")
	obj.SetProperty("a", 4, SignalTypeNone)
	obj.SetProperty("b", "banana", SignalTypeNone)
	obj.SetProperty("d", 5, SignalTypeNone)

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

	obj.SetProperty("d", 7, SignalTypeInvalidated)
	u = assertUpdated(t, w, "On signal after move")
	require.Equal(t, PropertiesChange{"d": {5, 7}}, u)

	obj1 := srv1.Object("/org/i3barista/objects/NotFoo", "org.i3barista.Service")
	obj1.SetProperty("d", 8, SignalTypeChanged)
	assertNotUpdated(t, w, "Different object in same service updated")

	w.AddSignalHandler("Signal",
		func(s *Signal, f Fetcher) map[string]interface{} {
			val, _ := f("foo")
			return map[string]interface{}{
				"fromSignal": s.Body[0],
				"fetched":    val,
			}
		})
	w.FetchOnSignal("nosignal")

	obj.Emit("Signal", 5)
	u = assertUpdated(t, w, "On signal handler")
	require.Equal(t, PropertiesChange{
		"fromSignal": {nil, 5},
		"fetched":    {nil, nil},
	}, u)

	obj.SetProperty("foo", 0, SignalTypeNone)
	obj.Emit("Signal", 8)
	u = assertUpdated(t, w, "On signal handler")
	require.Equal(t, PropertiesChange{
		"fromSignal": {5, 8},
		"fetched":    {nil, 0},
	}, u)

	obj.SetProperty("nosignal", "changed", SignalTypeNone)
	obj.SetProperty("a", 5, SignalTypeChanged)

	u = assertUpdated(t, w, "On change")
	require.Equal(t, PropertiesChange{
		"a":        {4, 5},
		"nosignal": {nil, "changed"},
	}, u)

	w.Add("org.i3barista.OtherService.Property")
	assertNotUpdated(t, w, "Unset property added")
	srv1.Object("/org/i3barista/objects/Foo", "org.i3barista.OtherService").
		SetProperty("Property", 4, SignalTypeInvalidated)
	u = assertUpdated(t, w, "Fully qualified property name")
	require.Equal(t, [2]interface{}{nil, 4},
		u["org.i3barista.OtherService.Property"],
		"Fully qualified property in change event")

	srv1.RemoveName("org.i3barista.services.FooService")
	u = assertUpdated(t, w, "On service disconnect")
	require.Equal(t, PropertiesChange{
		"a":                                   {5, nil},
		"b":                                   {"banana", nil},
		"d":                                   {7, nil},
		"fromSignal":                          {8, nil},
		"fetched":                             {0, nil},
		"nosignal":                            {"changed", nil},
		"org.i3barista.OtherService.Property": {4, nil},
	}, u, "All properties cleared")
	require.Empty(t, w.Get())

	obj.Emit("Signal", "c")
	assertNotUpdated(t, w, "After service disconnect")

	w.Fetch("manual")

	_, err = w.Call("Method", 3, 1, 4)
	require.Error(t, err, "On method call while disconnected")

	srv.AddName("org.i3barista.services.FooService")
	assertUpdated(t, w, "On service connect")
	require.Equal(t, map[string]interface{}{
		"a": 2,
		"c": "anotherstring",
		"d": 5,
	}, w.Get())

	obj = srv.Object("/org/i3barista/objects/Foo", "org.i3barista.Service")
	obj.SetProperty("manual", []string{"foo"}, SignalTypeChanged)
	assertNotUpdated(t, w, "non-signal property changed")

	require.Equal(t, map[string]interface{}{
		"a":      2,
		"c":      "anotherstring",
		"d":      5,
		"manual": []string{"foo"},
	}, w.Get(), "non-signal property included in fetch")

	w.FetchOnSignal("signalOrFetch").Fetch("signalOrFetch")
	obj.SetProperty("signalOrFetch", 2, SignalTypeChanged)
	u = assertUpdated(t, w, "property changed")
	require.Equal(t, PropertiesChange{
		"signalOrFetch": {nil, 2},
	}, u)

	obj.SetProperty("signalOrFetch", 3, SignalTypeNone)
	assertNotUpdated(t, w, "property changed without signal")

	obj.SetProperty("a", 5, SignalTypeChanged)
	u = assertUpdated(t, w, "property changed with signal")
	require.Equal(t, PropertiesChange{
		"signalOrFetch": {2, 3},
		"a":             {2, 5},
	}, u)

	w.Unsubscribe()
	require.Empty(t, w.Get(), "After Unsubscribe")

	obj.SetProperty("a", 4, SignalTypeChanged)
	assertNotUpdated(t, w, "after Unsubscribe")

	obj.Emit("Signal", "foo")
	assertNotUpdated(t, w, "after Unsubscribe")
}
