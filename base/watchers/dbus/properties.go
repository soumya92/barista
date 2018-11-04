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
	"sync"

	"github.com/godbus/dbus"
)

// PropertiesChange is emitted on PropertiesWatcher.Updates whenever any
// properties change. The key is the name of the property changed, and the value
// is a pair of interface{} values: {oldValue, newValue}.
type PropertiesChange map[string][2]interface{}

// Signal re-exports dbus.Signal to avoid namespace clashes for consumers.
type Signal = dbus.Signal

// Fetcher represents a function that returns the current value of a property
// and any error that ocurred while fetching it.
type Fetcher func(string) (interface{}, error)

// PropertiesWatcher is a watcher for the properties of a DBus object. It
// provides update notifications and the ability to map custom signals to
// property changes.
type PropertiesWatcher struct {
	Updates  <-chan PropertiesChange
	onChange chan<- PropertiesChange

	conn   dbusConn
	obj    dbus.BusObject
	dbusCh chan *Signal

	service   string
	object    dbus.ObjectPath
	iface     string
	propNames map[string]bool

	mu sync.RWMutex

	owner    string
	props    map[string]interface{} // Extracted from dbus.Variant values.
	handlers map[string]func(*Signal, Fetcher) map[string]interface{}
	signals  []dbusName
}

// Get returns the latest snapshot of all registered properties.
func (p *PropertiesWatcher) Get() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	r := map[string]interface{}{}
	for k, v := range p.props {
		r[k] = v
	}
	return r
}

// AddSignalHandler adds a signal handler for a signal emitted by the interface
// being watched that updates properties without emitting a PropertiesChanged
// signal (e.g. mpris 'Seeked').
// The handler function should return a map of all properties that have changed.
func (p *PropertiesWatcher) AddSignalHandler(
	name string,
	handler func(*Signal, Fetcher) map[string]interface{},
) *PropertiesWatcher {
	p.mu.Lock()
	defer p.mu.Unlock()
	nm := makeDbusName(name)
	if nm.iface == "" {
		nm.iface = p.iface
	}
	p.signals = append(p.signals, nm)
	if p.owner != "" {
		nm.addMatch(p.conn, p.matchOptions()...)
	}
	p.handlers[nm.String()] = handler
	return p
}

// Call calls a DBus method on the object being watched and returns the result.
// This method will deadlock if called from within a signal handler.
func (p *PropertiesWatcher) Call(name string, args ...interface{}) ([]interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.owner == "" {
		return nil, errors.New("Disconnected")
	}
	c := p.obj.Call(expand(p.iface, name), 0, args...)
	return c.Body, c.Err
}

// Unsubscribe clears all subscriptions and internal state. The watcher cannot
// be used after calling this method. Usually `defer`d when creating a watcher.
func (p *PropertiesWatcher) Unsubscribe() {
	p.conn.RemoveSignal(p.dbusCh)
	p.conn.Close()
	p.mu.RLock()
	defer p.mu.RUnlock()
	p.props = nil
}

func (p *PropertiesWatcher) listen() {
	for sig := range p.dbusCh {
		if sig.Name == nameOwnerChanged.String() {
			p.ownerChanged(sig.Body[2].(string), true)
		} else {
			p.handleSignal(sig)
		}
	}
}

func (p *PropertiesWatcher) handleSignal(sig *Signal) {
	p.mu.Lock()
	defer p.mu.Unlock()
	// This is fine, we should only get signals for which handlers have been
	// added. This can only panic if the internal state is somehow inconsistent.
	newProps := p.handlers[sig.Name](sig, p.fetch)
	if len(newProps) == 0 {
		return
	}
	ch := PropertiesChange{}
	for k, v := range newProps {
		ch[k] = [2]interface{}{p.props[k], v}
		p.props[k] = v
	}
	p.onChange <- ch
}

func (p *PropertiesWatcher) fetch(propName string) (interface{}, error) {
	// p.obj != nil because all signal matches are filtered by owner. If there
	// is no owner, there will also be no matches.
	val, err := p.obj.GetProperty(expand(p.iface, propName))
	return val.Value(), err
}

func (p *PropertiesWatcher) matchOptions() []dbus.MatchOption {
	return []dbus.MatchOption{
		dbus.WithMatchOption("sender", p.owner),
		dbus.WithMatchOption("path", string(p.object)),
	}
}

func (p *PropertiesWatcher) ownerChanged(owner string, signal bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.owner != "" {
		m := p.matchOptions()
		propsChanged.removeMatch(p.conn, m...)
		for _, s := range p.signals {
			s.removeMatch(p.conn, m...)
		}
	}
	p.owner = owner
	newProps := map[string]interface{}{}
	if p.owner != "" {
		p.obj = p.conn.Object(p.service, p.object)
		for propName := range p.propNames {
			if val, err := p.fetch(propName); err == nil {
				newProps[propName] = val
			}
		}
		m := p.matchOptions()
		propsChanged.addMatch(p.conn, m...)
		for _, s := range p.signals {
			s.addMatch(p.conn, m...)
		}
	}
	if signal {
		ch := PropertiesChange{}
		for k, v := range p.props {
			ch[k] = [2]interface{}{v, newProps[k]}
		}
		for k, v := range newProps {
			_, ok := ch[k]
			if !ok {
				ch[k] = [2]interface{}{nil, v}
			}
		}
		p.onChange <- ch
	}
	p.props = newProps
}

func (p *PropertiesWatcher) propChangeHandler(sig *Signal, fetch Fetcher) map[string]interface{} {
	m := sig.Body[1].(map[string]dbus.Variant)
	r := map[string]interface{}{}
	for k, v := range m {
		k = shorten(p.iface, k)
		if p.propNames[k] {
			r[k] = v.Value()
		}
	}
	invalidated, _ := sig.Body[2].([]string)
	for _, k := range invalidated {
		k = shorten(p.iface, k)
		if !p.propNames[k] {
			continue
		}
		if v, err := fetch(k); err == nil {
			r[k] = v
		}
	}
	return r
}

// WatchProperties constructs a DBus properties watcher for the given object and
// interface, using a specified bus and service name. The list of properties is
// further used to filter events, as well as to fetch initial data when the
// watcher is constructed. Watchers must be cleaned up by calling Unsubscribe.
func WatchProperties(
	busType BusType,
	service string,
	object dbus.ObjectPath,
	iface string,
	properties []string,
) *PropertiesWatcher {
	conn := busType()
	updates := make(chan PropertiesChange)
	w := &PropertiesWatcher{
		conn:      conn,
		props:     map[string]interface{}{},
		dbusCh:    make(chan *Signal, 10),
		service:   service,
		object:    object,
		iface:     iface,
		propNames: map[string]bool{},
		Updates:   updates,
		onChange:  updates,
	}
	for _, p := range properties {
		w.propNames[p] = true
	}
	w.handlers = map[string]func(*Signal, Fetcher) map[string]interface{}{
		propsChanged.String(): w.propChangeHandler,
	}
	var owner string
	if err := getNameOwner.call(conn, service).Store(&owner); err == nil {
		w.ownerChanged(owner, false)
	}
	nameOwnerChanged.addMatch(conn, dbus.WithMatchOption("arg0", service))
	w.conn.Signal(w.dbusCh)
	go w.listen()
	return w
}
