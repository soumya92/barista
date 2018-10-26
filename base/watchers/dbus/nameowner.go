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

// Package dbus provides watchers that notify when dbus name owners or object
// properties change.
package dbus // import "barista.run/base/watchers/dbus"

import (
	"fmt"
	"strings"
	"sync"

	"barista.run/base/notifier"
	"github.com/godbus/dbus"
)

// NameOwnerWatcher is a watcher for a single or wildcard service name owner.
// It notifies on any changes to names of interest, and provides methods to get
// the current owner(s) of those names.
type NameOwnerWatcher struct {
	C <-chan struct{}

	conn   *dbus.Conn
	dbusCh chan *dbus.Signal

	notifyFn func()

	owners   map[string]string
	ownersMu sync.RWMutex
}

// GetOwner gets an owner of a service name of interest. For an exact watcher,
// this returns the owner of the service name (or empty if no owner), for a
// wildcard watcher it returns a random owner from all that match.
// Even with wildcard watcher, this method can be used to get the name of the
// lone owner of the service name.
func (n *NameOwnerWatcher) GetOwner() string {
	n.ownersMu.RLock()
	defer n.ownersMu.RUnlock()
	for _, v := range n.owners {
		return v
	}
	return ""
}

// GetOwners returns a map of service names to owners for all services that
// match the watcher criterion.
func (n *NameOwnerWatcher) GetOwners() map[string]string {
	n.ownersMu.RLock()
	defer n.ownersMu.RUnlock()
	result := map[string]string{}
	for k, v := range n.owners {
		result[k] = v
	}
	return result
}

// Unsubscribe clears all subscriptions and internal state. The watcher cannot
// be used after calling this method. Usually `defer`d when creating a watcher.
func (n *NameOwnerWatcher) Unsubscribe() {
	n.conn.RemoveSignal(n.dbusCh)
	n.conn.Close()
	n.ownersMu.RLock()
	defer n.ownersMu.RUnlock()
	n.owners = map[string]string{}
}

func (n *NameOwnerWatcher) listen() {
	n.conn.Signal(n.dbusCh)
	for sig := range n.dbusCh {
		name := sig.Body[0].(string)
		newOwner := sig.Body[2].(string)
		n.ownersMu.Lock()
		if len(newOwner) == 0 {
			delete(n.owners, name)
		} else {
			n.owners[name] = newOwner
		}
		n.ownersMu.Unlock()
		n.notifyFn()
	}
}

func watchNameOwner(name string, wildcard bool) (*NameOwnerWatcher, error) {
	sessionBus, err := dbus.SessionBusPrivate()
	if err == nil {
		err = sessionBus.Auth(nil)
	}
	if err == nil {
		err = sessionBus.Hello()
	}
	if err != nil {
		return nil, err
	}
	bus := sessionBus.BusObject()
	watcher := &NameOwnerWatcher{
		conn:   sessionBus,
		owners: map[string]string{},
		dbusCh: make(chan *dbus.Signal, 10),
	}
	watcher.notifyFn, watcher.C = notifier.New()
	var names []string
	bus.Call("ListNames", 0).Store(&names)
	for _, n := range names {
		if nameMatch(n, name, wildcard) {
			var owner string
			if err := bus.Call("GetNameOwner", 0, n).Store(&owner); err == nil {
				watcher.owners[n] = owner
			}
		}
	}
	matchString := "type='signal',interface='org.freedesktop.DBus',member='NameOwnerChanged'"
	if wildcard {
		matchString += fmt.Sprintf(",arg0namespace='%s'", name)
	} else {
		matchString += fmt.Sprintf(",arg0='%s'", name)
	}
	bus.Call("AddMatch", 0, matchString)
	go watcher.listen()
	return watcher, nil
}

func nameMatch(val, search string, wildcard bool) bool {
	if !wildcard {
		return val == search
	}
	return val == search || strings.HasPrefix(val, search+".")
}

// WatchNameOwner creates a watcher for exactly the name given.
func WatchNameOwner(name string) (*NameOwnerWatcher, error) {
	return watchNameOwner(name, false)
}

// WatchNameOwners creates a watcher for any names within the 'namespace' given.
// For example, 'com.example.backend1' will notify for
// 'com.example.backend1.foo', 'com.example.backend1.foo.bar', and
// 'com.example.backend1' itself.
// All matching names and their owners can be retrieved using GetOwners().
func WatchNameOwners(pattern string) (*NameOwnerWatcher, error) {
	return watchNameOwner(pattern, true)
}
