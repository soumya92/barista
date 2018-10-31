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

	"github.com/godbus/dbus"
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
	defer t.bus.mu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.names[name] {
		return // otherwise deadlock trying to remove name from previous owner.
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
	go t.bus.busObj.Emit(nameOwnerChanged.String(), name, oldOwner, t.id)
}

// RemoveName unregisters the service for the given well-known name.
func (t *TestBusService) RemoveName(name string) {
	t.bus.mu.Lock()
	defer t.bus.mu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.names[name] {
		return
	}
	delete(t.bus.services, name)
	delete(t.names, name)
	go t.bus.busObj.Emit(nameOwnerChanged.String(), name, t.id, "")
}

// Unregister unregisters the service from the bus completely. The service and
// all associated objects are unusable after this.
func (t *TestBusService) Unregister() {
	if !atomic.CompareAndSwapInt64(&t.destroyed, 0, 1) {
		panic("Unregistering already unregistered service")
	}
	t.bus.mu.Lock()
	defer t.bus.mu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	for n := range t.names {
		delete(t.bus.services, n)
		go t.bus.busObj.Emit(nameOwnerChanged.String(), n, t.id, "")
	}
	t.id = ""
	t.names = nil
	t.objects = nil
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
		if dest == "" {
			dest = t.anyName()
		}
		o = &testBusObject{
			svc: t, dest: dest, path: path,
			props: map[string]interface{}{},
			calls: map[string]func(...interface{}) ([]interface{}, error){},
		}
		t.objects[path] = o
	}
	return &TestBusObject{o, nil /* conn set by caller */}
}
