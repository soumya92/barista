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
	"sync"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
)

// TestBusService represents a test service on the bus.
type TestBusService struct {
	mu        sync.Mutex
	destroyed int64 // atomic bool

	bus     *TestBus
	id      string
	names   map[string]bool
	objects map[dbus.ObjectPath]*testBusObject
}

// AddName registers the service for the given well-known name.
func (t *TestBusService) AddName(name string) {
	t.bus.mu.Lock()
	t.mu.Lock()
	oldOwner, ok := t.addNameLocked(name)
	busObj := t.bus.busObj
	t.mu.Unlock()
	t.bus.mu.Unlock()
	if ok {
		busObj.Emit(nameOwnerChanged.String(), name, oldOwner, t.id)
	}
}

func (t *TestBusService) addNameLocked(name string) (string, bool) {
	if t.names[name] {
		// Prevent deadlock trying to remove name from previous owner (self).
		return "", false
	}
	oldOwner := ""
	if prev := t.bus.services[name]; prev != nil {
		oldOwner = prev.id
		prev.mu.Lock()
		delete(prev.names, name)
		prev.mu.Unlock()
	}
	t.bus.services[name] = t
	t.names[name] = true
	return oldOwner, true
}

// RemoveName unregisters the service for the given well-known name.
func (t *TestBusService) RemoveName(name string) {
	t.bus.mu.Lock()
	t.mu.Lock()
	ok := t.removeNameLocked(name)
	busObj := t.bus.busObj
	t.mu.Unlock()
	t.bus.mu.Unlock()
	if ok {
		busObj.Emit(nameOwnerChanged.String(), name, t.id, "")
	}
}

func (t *TestBusService) removeNameLocked(name string) bool {
	if !t.names[name] {
		return false
	}
	delete(t.bus.services, name)
	delete(t.names, name)
	return true
}

// Unregister unregisters the service from the bus completely. The service and
// all associated objects are unusable after this.
func (t *TestBusService) Unregister() {
	if !atomic.CompareAndSwapInt64(&t.destroyed, 0, 1) {
		panic("Unregistering already unregistered service")
	}
	t.bus.mu.Lock()
	t.mu.Lock()
	id, names := t.unregisterLocked()
	busObj := t.bus.busObj
	t.mu.Unlock()
	t.bus.mu.Unlock()
	for _, n := range names {
		busObj.Emit(nameOwnerChanged.String(), n, id, "")
	}
}

func (t *TestBusService) unregisterLocked() (id string, names []string) {
	for n := range t.names {
		delete(t.bus.services, n)
		names = append(names, n)
	}
	id = t.id
	t.id = ""
	t.names = nil
	t.objects = nil
	return id, names
}

// checkRegistered panics if the service has been unregistered.
func (t *TestBusService) checkRegistered() {
	if atomic.LoadInt64(&t.destroyed) == 1 {
		panic("trying to use object from unregistered service")
	}
}

// anyName returns a registered name, or an empty string if none are available.
func (t *TestBusService) anyName() string {
	for n := range t.names {
		return n
	}
	return ""
}

// Object returns a test object on the service at the given path. If non-empty,
// dest is used to override the destination interface for the object.
func (t *TestBusService) Object(path dbus.ObjectPath, dest string) *TestBusObject {
	t.mu.Lock()
	defer t.mu.Unlock()
	o, ok := t.objects[path]
	if !ok {
		o = &testBusObject{
			svc: t, path: path,
			props: map[string]interface{}{},
			calls: map[string]func(...interface{}) ([]interface{}, error){},
		}
		t.objects[path] = o
	}
	if dest == "" {
		dest = t.anyName()
	}
	return &TestBusObject{o, dest, nil /* conn set by caller */}
}
