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

	"github.com/godbus/dbus/v5"
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

	service string
	object  dbus.ObjectPath
	iface   string
	props   map[string]propertyUpdateType

	mu sync.RWMutex

	owner   string
	signals map[dbusName]func(*Signal, Fetcher) map[string]interface{}

	lastProps map[string]interface{} // Extracted from dbus.Variant values.
}

// Get returns the latest snapshot of all registered properties.
func (p *PropertiesWatcher) Get() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	r := map[string]interface{}{}
	for k, v := range p.lastProps {
		r[k] = v
	}
	if p.owner == "" {
		return r
	}
	for prop, u := range p.props {
		if u&updateTypeManual == 0 {
			continue
		}
		if val, err := p.fetch(prop); err == nil {
			r[prop] = val
		}
	}
	return r
}

// Add specifies properties to watch, relying on the PropertiesChanged signal
// to update their value.
func (p *PropertiesWatcher) Add(props ...string) *PropertiesWatcher {
	return p.addProperties(updateTypeSignal, props)
}

// FetchOnSignal specifies additional properties to fetch when the object emits
// PropertiesChanged. This can be useful for tracking computed properties if
// their value only changes when other properties also change.
// These properties will be included in emitted PropertiesChange values.
func (p *PropertiesWatcher) FetchOnSignal(props ...string) *PropertiesWatcher {
	return p.addProperties(updateTypeFetchOnSignal, props)
}

// Fetch specifies additional properties to fetch each time the full set of
// properties is requested via Get().
func (p *PropertiesWatcher) Fetch(props ...string) *PropertiesWatcher {
	return p.addProperties(updateTypeManual, props)
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
	if p.owner != "" {
		nm.addMatch(p.conn, p.matchOptions()...)
	}
	p.signals[nm] = handler
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
	p.mu.Lock()
	defer p.mu.Unlock()
	p.conn.RemoveSignal(p.dbusCh)
	p.conn.Close()
	p.lastProps = nil
	p.owner = ""
}

type propertyUpdateType int

const (
	// updateTypeSignal indicates that the property should be updated based on
	// the PropertiesChanged signals sent by DBus.
	updateTypeSignal propertyUpdateType = 1 << 0
	// updateTypeFetch indicates that the property should be fetched on
	// receiving PropertiesChanged signals, but the property itself may not be
	// included in the signal body.
	updateTypeFetch propertyUpdateType = 1 << 1
	// updateTypeFetchOnSignal indicates that the property should be fetched on
	// receiving PropertiesChanged signals, but any value present in the signal
	// body should be preferred.
	updateTypeFetchOnSignal propertyUpdateType = updateTypeSignal | updateTypeFetch
	// updateTypeManual indicates that the property should be fetched on each
	// call to Get(). Manually updated properties will never trigger change
	// notifications.
	updateTypeManual propertyUpdateType = 1 << 2
)

func (p *PropertiesWatcher) addProperties(updateType propertyUpdateType, props []string) *PropertiesWatcher {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, prop := range props {
		p.props[prop] = p.props[prop] | updateType
		if p.owner == "" {
			continue
		}
		if val, err := p.fetch(prop); err == nil {
			p.lastProps[prop] = val
		}
	}
	return p
}

func (p *PropertiesWatcher) listen() {
	for sig := range p.dbusCh {
		if sig.Name == nameOwnerChanged.String() {
			p.ownerChanged(sig.Body[2].(string))
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
	newProps := p.signals[makeDbusName(sig.Name)](sig, p.fetch)
	if len(newProps) == 0 {
		return
	}
	ch := PropertiesChange{}
	for k, v := range newProps {
		ch[k] = [2]interface{}{p.lastProps[k], v}
		p.lastProps[k] = v
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

func (p *PropertiesWatcher) ownerChanged(owner string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.owner != "" {
		m := p.matchOptions()
		for s := range p.signals {
			s.removeMatch(p.conn, m...)
		}
	}
	p.owner = owner
	if p.owner == "" {
		ch := PropertiesChange{}
		for k, oldVal := range p.lastProps {
			ch[k] = [2]interface{}{oldVal, nil}
			delete(p.lastProps, k)
		}
		if len(ch) > 0 {
			p.onChange <- ch
		}
		return
	}
	p.obj = p.conn.Object(p.service, p.object)
	m := p.matchOptions()
	for s := range p.signals {
		s.addMatch(p.conn, m...)
	}
	if len(p.props) == 0 {
		return
	}
	ch := PropertiesChange{}
	for k := range p.props {
		oldVal, ok := p.lastProps[k]
		newVal, err := p.fetch(k)
		if err == nil {
			p.lastProps[k] = newVal
		} else {
			delete(p.lastProps, k)
		}
		if err == nil || ok {
			ch[k] = [2]interface{}{oldVal, newVal}
		}
	}
	p.onChange <- ch
}

func (p *PropertiesWatcher) propChangeHandler(sig *Signal, fetch Fetcher) map[string]interface{} {
	m := sig.Body[1].(map[string]dbus.Variant)
	r := map[string]interface{}{}
	for k, v := range m {
		k = shorten(p.iface, k)
		if p.props[k]&updateTypeSignal != 0 {
			r[k] = v.Value()
		}
	}
	invalidated, _ := sig.Body[2].([]string)
	for _, k := range invalidated {
		k = shorten(p.iface, k)
		if p.props[k]&updateTypeSignal == 0 {
			continue
		}
		if v, err := fetch(k); err == nil {
			r[k] = v
		}
	}
	if len(r) == 0 {
		return r
	}
	for k, v := range p.props {
		if v&updateTypeFetch == 0 {
			continue
		}
		if _, ok := r[k]; ok {
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
func WatchProperties(busType BusType, service string, object string, iface string) *PropertiesWatcher {
	conn := busType()
	updates := make(chan PropertiesChange, 10)
	w := &PropertiesWatcher{
		Updates:   updates,
		onChange:  updates,
		conn:      conn,
		dbusCh:    make(chan *Signal, 10),
		service:   service,
		object:    dbus.ObjectPath(object),
		iface:     iface,
		props:     map[string]propertyUpdateType{},
		signals:   map[dbusName]func(*Signal, Fetcher) map[string]interface{}{},
		lastProps: map[string]interface{}{},
	}
	w.signals[propsChanged] = w.propChangeHandler
	var owner string
	if err := getNameOwner.call(conn, service).Store(&owner); err == nil {
		w.ownerChanged(owner)
	}
	nameOwnerChanged.addMatch(conn, dbus.WithMatchOption("arg0", service))
	w.conn.Signal(w.dbusCh)
	go w.listen()
	return w
}
