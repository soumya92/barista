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
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

// NameOwnerChange is emitted on NameOwnerWatcher.Updates whenever any name
// is acquired or released. The Owner is the new owner of the name, and is empty
// if the name was released.
type NameOwnerChange struct {
	Name, Owner string
}

// NameOwnerWatcher is a watcher for a single or wildcard service name owner.
// It notifies on any changes to names of interest, and provides methods to get
// the current owner(s) of those names.
type NameOwnerWatcher struct {
	Updates  <-chan NameOwnerChange
	onChange chan<- NameOwnerChange

	conn   dbusConn
	dbusCh chan *dbus.Signal
	match  dbus.MatchOption

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
	n.owners = nil
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
		n.onChange <- NameOwnerChange{name, newOwner}
	}
}

// WatchNameOwner creates a watcher for exactly the name given.
func WatchNameOwner(busType BusType, name string) *NameOwnerWatcher {
	return watchNameOwner(
		busType,
		func(search string) bool { return search == name },
		dbus.WithMatchOption("arg0", name),
	)
}

// WatchNameOwners creates a watcher for any names within the 'namespace' given.
// For example, 'com.example.backend1' will notify for
// 'com.example.backend1.foo', 'com.example.backend1.foo.bar', and
// 'com.example.backend1' itself.
// All matching names and their owners can be retrieved using GetOwners().
func WatchNameOwners(busType BusType, pattern string) *NameOwnerWatcher {
	return watchNameOwner(
		busType,
		func(search string) bool {
			return search == pattern || strings.HasPrefix(search, pattern+".")
		},
		dbus.WithMatchOption("arg0namespace", pattern),
	)
}

func watchNameOwner(
	busType BusType, matcher func(string) bool, matchOption dbus.MatchOption,
) *NameOwnerWatcher {
	conn := busType()
	updates := make(chan NameOwnerChange, 1)
	watcher := &NameOwnerWatcher{
		conn:     conn,
		owners:   map[string]string{},
		dbusCh:   make(chan *dbus.Signal, 10),
		onChange: updates,
		Updates:  updates,
	}
	var names []string
	listNames.call(conn).Store(&names)
	for _, n := range names {
		if !matcher(n) {
			continue
		}
		var owner string
		if err := getNameOwner.call(conn, n).Store(&owner); err == nil {
			watcher.owners[n] = owner
		}
	}
	nameOwnerChanged.addMatch(conn, matchOption)
	go watcher.listen()
	return watcher
}
