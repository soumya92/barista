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
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// testBusObject represents an object on the test bus.
type testBusObject struct {
	mu sync.Mutex

	svc   *TestBusService
	path  dbus.ObjectPath
	props map[string]interface{}
	calls map[string]func(...interface{}) ([]interface{}, error)
	// elseCall: fallback when calls[method] is not defined.
	eCall func(string, ...interface{}) ([]interface{}, error)
}

// TestBusObject represents a connection to an object on the test bus.
type TestBusObject struct {
	*testBusObject
	dest string
	conn *testBusConnection
}

// Call calls a method with and waits for its reply.
func (t *TestBusObject) Call(method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	t.check()
	method = expand(t.dest, method)
	call := &dbus.Call{
		Destination: t.dest,
		Path:        t.path,
		Method:      method,
		Args:        args,
		Done:        make(chan *dbus.Call, 1),
	}
	call.Done <- call
	t.mu.Lock()
	defer t.mu.Unlock()
	h, ok := t.calls[method]
	if !ok && t.eCall != nil {
		h = func(args ...interface{}) ([]interface{}, error) {
			return t.eCall(method, args...)
		}
	}
	if h == nil {
		call.Err = errors.New("No such method: " + method)
	} else {
		call.Body, call.Err = h(args...)
	}
	return call
}

// CallWithContext acts like Call but takes a context.
func (t *TestBusObject) CallWithContext(ctx context.Context, method string, flags dbus.Flags, args ...interface{}) *dbus.Call {
	return t.Call(method, flags, args...)
}

// Go calls a method with the given arguments asynchronously.
func (t *TestBusObject) Go(method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
	go func() {
		// Halfway between the positive (10ms) and negative (1s) timeouts.
		time.Sleep(505 * time.Millisecond)
		ch <- t.Call(method, flags, args...)
	}()
	return nil
}

// GoWithContext acts like Go but takes a context.
func (t *TestBusObject) GoWithContext(ctx context.Context, method string, flags dbus.Flags, ch chan *dbus.Call, args ...interface{}) *dbus.Call {
	return t.Go(method, flags, ch, args...)
}

// matchCallResult creates a dbus.Call result for Add/RemoveMatch.
func matchCallResult(method string, err error) *dbus.Call {
	c := &dbus.Call{
		Destination: bus,
		Path:        busPath,
		Method:      expand(bus, method),
		Args:        []interface{}{"should not matter"},
		Done:        make(chan *dbus.Call, 1),
		Err:         err,
	}
	c.Done <- c
	return c
}

// AddMatchSignal subscribes BusObject to signals from specified interface and
// method with the given filters.
func (t *TestBusObject) AddMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	name := iface + "." + member
	t.check()
	optMap := dbusMatchOptionMap(options)
	for k := range optMap {
		if k == "path" || k == "path_namespace" || k == "sender" {
			continue
		}
		if strings.HasPrefix(k, "arg") {
			continue
		}
		return matchCallResult("AddMatch", errors.New("Unsupported match type: "+k))
	}
	t.conn.mu.Lock()
	defer t.conn.mu.Unlock()
	t.conn.matches[name] = append(t.conn.matches[name], optMap)
	return matchCallResult("AddMatch", nil)
}

// RemoveMatchSignal unsubscribes BusObject from signals from specified
// interface and method with the given filters.
func (t *TestBusObject) RemoveMatchSignal(iface, member string, options ...dbus.MatchOption) *dbus.Call {
	name := iface + "." + member
	t.check()
	t.conn.mu.Lock()
	defer t.conn.mu.Unlock()
	ms := t.conn.matches[name]
	optMap := dbusMatchOptionMap(options)
	for i, m := range ms {
		if reflect.DeepEqual(m, optMap) {
			t.conn.matches[name] = append(ms[:i], ms[i+1:]...)
			return matchCallResult("RemoveMatch", nil)
		}
	}
	return matchCallResult("RemoveMatch", errors.New("Match not found"))
}

// GetProperty returns the value of a named property.
func (t *TestBusObject) GetProperty(p string) (dbus.Variant, error) {
	t.check()
	t.mu.Lock()
	defer t.mu.Unlock()
	if val, ok := t.props[p]; ok {
		return dbus.MakeVariant(val), nil
	}
	return dbus.Variant{}, errors.New("No such property: " + p)
}

// StoreProperty stores the value of a named property into a given pointer.
func (t *TestBusObject) StoreProperty(p string, dest interface{}) error {
	val, err := t.GetProperty(p)
	if err == nil {
		err = dbus.Store([]interface{}{val}, dest)
	}
	return err
}

// Destination returns the destination that calls on are sent to.
func (t *TestBusObject) Destination() string {
	t.check()
	return t.dest
}

// Path returns the path that calls are sent to.
func (t *TestBusObject) Path() dbus.ObjectPath {
	t.check()
	return t.path
}

// SignalType controls the type of signal sent on properties change
type SignalType byte

const (
	// SignalTypeNone does not emit any signal on properties change.
	SignalTypeNone SignalType = iota
	// SignalTypeChanged emits a PropertiesChanged signal with values for each
	// modified property in changed_properties.
	SignalTypeChanged
	// SignalTypeInvalidated emits a PropertiesChanged signal with only the
	// property names in invalidated_properties.
	SignalTypeInvalidated
)

// SetProperty sets a property of the test object.
func (t *TestBusObject) SetProperty(prop string, value interface{}) error {
	t.SetPropertyForTest(prop, value, SignalTypeChanged)
	return nil
}

// SetPropertyForTest sets a property of the test object. The signal type parameter
// controls whether a "PropertiesChanged" signal is automatically emitted, and
// what form the emitted signal takes.
func (t *TestBusObject) SetPropertyForTest(prop string, value interface{}, signalType SignalType) {
	t.SetProperties(map[string]interface{}{prop: value}, signalType)
}

// SetProperties sets multiple properties of the test object. The signal type
// controls whether a "PropertiesChanged" signal is automatically emitted, and
// what form the emitted signal takes.
func (t *TestBusObject) SetProperties(props map[string]interface{}, signalType SignalType) {
	t.check()
	t.mu.Lock()
	defer t.mu.Unlock()
	for k, v := range props {
		t.props[expand(t.dest, k)] = v
	}
	chg := map[string]dbus.Variant{}
	inv := []string{}
	switch signalType {
	case SignalTypeNone:
		return
	case SignalTypeInvalidated:
		for k := range props {
			inv = append(inv, expand(t.dest, k))
		}
	case SignalTypeChanged:
		for k, v := range props {
			chg[expand(t.dest, k)] = dbus.MakeVariant(v)
		}
	}
	go t.Emit(propsChanged.String(), t.dest, chg, inv)
}

// On sets up a function to be called when the given named method is invoked,
// and returns the result of the function to the method caller.
func (t *TestBusObject) On(method string, do func(...interface{}) ([]interface{}, error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.calls[expand(t.dest, method)] = do
}

// OnElse sets the function to be called for all methods that don't have a
// separate On(method, ...) definition.
func (t *TestBusObject) OnElse(do func(string, ...interface{}) ([]interface{}, error)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.eCall = do

}

// Emit emits a signal on the test bus, dispatching it to relevant listeners.
func (t *TestBusObject) Emit(name string, args ...interface{}) {
	name = expand(t.dest, name)
	t.svc.bus.emit(name, t.svc.id, t.path, args...)
}

// check panics if the service is unregistered or the connection is closed.
func (t *TestBusObject) check() {
	t.svc.checkRegistered()
	if t.conn != nil {
		// conn can be nil if the object is not associated with a connection,
		// e.g. obtained directly from a TestBusService.
		t.conn.checkOpen()
	}
}
