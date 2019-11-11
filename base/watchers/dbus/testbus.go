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
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"barista.run/logging"

	"github.com/godbus/dbus/v5"
)

// TestBus represents a mock DBus interface for testing.
type TestBus struct {
	mu sync.Mutex

	busObj      *TestBusObject
	nextID      int
	services    map[string]*TestBusService
	connections map[*testBusConnection]bool
}

// newTestBus constructs a new test bus, priming it with the name owner methods.
func newTestBus() *TestBus {
	t := &TestBus{
		services:    map[string]*TestBusService{},
		connections: map[*testBusConnection]bool{},
	}
	t.RegisterService(bus)
	t.busObj = t.Object(bus, busPath)
	t.busObj.On("ListNames", func(args ...interface{}) ([]interface{}, error) {
		names := []string{}
		t.mu.Lock()
		defer t.mu.Unlock()
		for n := range t.services {
			names = append(names, n)
		}
		return []interface{}{names}, nil
	})
	t.busObj.On("GetNameOwner", func(args ...interface{}) ([]interface{}, error) {
		nm := args[0].(string)
		t.mu.Lock()
		defer t.mu.Unlock()
		svc := t.services[nm]
		if svc == nil {
			return nil, errors.New("No such service")
		}
		return []interface{}{svc.id}, nil
	})
	return t
}

// BusObject returns an object representing the bus itself.
func (t *TestBus) BusObject() *TestBusObject {
	return t.busObj
}

// Object returns the object at a given path of the specified service.
func (t *TestBus) Object(dest string, path dbus.ObjectPath) *TestBusObject {
	t.mu.Lock()
	defer t.mu.Unlock()
	svc := t.services[dest]
	if svc == nil {
		panic("No service for " + dest + " registered")
	}
	return svc.Object(path, dest)
}

// emit emits a signal to all interested connections.
func (t *TestBus) emit(name string, sender string, path dbus.ObjectPath, args ...interface{}) {
	logging.Log("%s (%s) Emit(%s, %+#v)", path, sender, name, args)
	signal := &dbus.Signal{
		Sender: sender,
		Path:   path,
		Name:   name,
		Body:   args,
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	for c := range t.connections {
		c.mu.Lock()
		if c.shouldSignal(name, sender, path, args) {
			for s := range c.signals {
				s <- signal
			}
		}
		c.mu.Unlock()
	}
}

// connect returns a new connection to the test bus.
func (t *TestBus) connect() *testBusConnection {
	conn := &testBusConnection{
		bus:     t,
		signals: map[chan<- *dbus.Signal]bool{},
		matches: map[string][]map[string]string{},
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	conn.busObj = &TestBusObject{t.BusObject().testBusObject, bus, conn}
	t.connections[conn] = true
	return conn
}

// testBusConnection represents a single connection to the test bus.
type testBusConnection struct {
	bus    *TestBus
	closed int64 // atomic bool

	mu      sync.Mutex
	busObj  *TestBusObject
	signals map[chan<- *dbus.Signal]bool
	matches map[string][]map[string]string
}

// Close closes the connection, rendering it unusable.
func (t *testBusConnection) Close() error {
	if !atomic.CompareAndSwapInt64(&t.closed, 0, 1) {
		return dbus.ErrClosed
	}
	t.bus.mu.Lock()
	delete(t.bus.connections, t)
	t.bus.mu.Unlock()
	t.mu.Lock()
	t.signals = nil
	t.matches = nil
	t.mu.Unlock()
	return nil
}

// BusObject returns an object representing the bus itself.
func (t *testBusConnection) BusObject() dbus.BusObject {
	t.checkOpen()
	return t.busObj
}

// Object returns the object identified by the given destination name and path.
func (t *testBusConnection) Object(dest string, path dbus.ObjectPath) dbus.BusObject {
	t.checkOpen()
	o := t.bus.Object(dest, path)
	o.conn = t
	return o
}

// RemoveSignal removes the given channel from the list of the registered channels.
func (t *testBusConnection) RemoveSignal(ch chan<- *dbus.Signal) {
	t.checkOpen()
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.signals, ch)
}

// Signal registers the given channel to be passed all received signal messages.
func (t *testBusConnection) Signal(ch chan<- *dbus.Signal) {
	t.checkOpen()
	t.mu.Lock()
	defer t.mu.Unlock()
	t.signals[ch] = true
}

// checkOpen panics if the connection has been closed.
func (t *testBusConnection) checkOpen() {
	if atomic.LoadInt64(&t.closed) == 1 {
		panic("trying to use closed connection")
	}
}

// shouldSignal returns true if the given signal would match any registered
// conditions for the connection.
func (t *testBusConnection) shouldSignal(name string, sender string, path dbus.ObjectPath, args []interface{}) bool {
	for _, cond := range t.matches[name] {
		matches := true
		for k, v := range cond {
			if !checkSignalCondition(k, v, sender, path, args) {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}

// RegisterService returns a new test service and optionally registers it for
// one or more well-known names.
func (t *TestBus) RegisterService(names ...string) *TestBusService {
	t.mu.Lock()
	svc, ownerChanges := t.registerServiceLocked(names...)
	busObj := t.busObj
	t.mu.Unlock()
	if busObj != nil {
		for n, chg := range ownerChanges {
			busObj.Emit(nameOwnerChanged.String(), n, chg[0], chg[1])
		}
	}
	return svc
}

func (t *TestBus) registerServiceLocked(names ...string) (service *TestBusService, ownerChanges map[string][2]string) {
	id := fmt.Sprintf(":%d", t.nextID)
	t.nextID++
	nameMap := map[string]bool{}
	for _, n := range names {
		nameMap[n] = true
	}
	svc := &TestBusService{
		bus: t, id: id, names: nameMap,
		objects: map[dbus.ObjectPath]*testBusObject{},
	}
	ownerChanges = map[string][2]string{}
	for n := range nameMap {
		oldOwner := ""
		if prev := t.services[n]; prev != nil {
			oldOwner = prev.id
			prev.mu.Lock()
			delete(prev.names, n)
			prev.mu.Unlock()
		}
		t.services[n] = svc
		ownerChanges[n] = [2]string{oldOwner, id}
	}
	return svc, ownerChanges
}

func checkSignalCondition(key, value string, sender string, path dbus.ObjectPath, args []interface{}) bool {
	pathStr := string(path)
	switch key {
	case "path":
		return pathStr == value
	case "path_namespace":
		return pathStr == value || strings.HasPrefix(pathStr, value+"/")
	case "sender":
		return sender == value
	}
	// TODO: Handle more than 10 arguments.
	argNum, _ := strconv.ParseInt(key[3:4], 10, 32)
	if len(args) <= int(argNum) {
		return false
	}
	var argVal string
	switch v := args[argNum].(type) {
	case string:
		argVal = v
	case dbus.ObjectPath:
		argVal = string(v)
	default:
		return false
	}
	switch key[4:] {
	case "namespace":
		return argVal == value || strings.HasPrefix(argVal, value+".")
	case "path":
		return argVal == value || strings.HasPrefix(argVal, value+"/")
	}
	return argVal == value
}

// dbusMatchOptionMap creates a map of key/value pairs from a list of
// dbus.MatchOptions using reflection to read the key/value fields.
func dbusMatchOptionMap(opts []dbus.MatchOption) map[string]string {
	m := map[string]string{}
	for _, o := range opts {
		opt := reflect.ValueOf(o)
		k := opt.FieldByName("key").String()
		v := opt.FieldByName("value").String()
		m[k] = v
	}
	return m
}
