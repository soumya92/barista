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
// properties change, and infrastructure for testing code that uses them.
package dbus // import "barista.run/base/watchers/dbus"

import (
	"strings"
	"sync/atomic"

	"github.com/godbus/dbus/v5"
)

// BusType represents a type of DBus connection: system, session, or test.
type BusType func() dbusConn

var (
	// Session connects to the current user's session DBus instance.
	Session BusType = sessionBus
	// System connects to the system-wide DBus instance.
	System BusType = systemBus
	// Test connects to a test bus. Use SetupTestBus() to set up a linked
	// controller for manipulating the test bus.
	Test BusType = testBus
)

func sessionBus() dbusConn { return connect(dbus.SessionBusPrivate()) }
func systemBus() dbusConn  { return connect(dbus.SystemBusPrivate()) }
func testBus() dbusConn    { return testBusInstance.Load().(*TestBus).connect() }

var testBusInstance atomic.Value // of *TestBus

// SetupTestBus sets up a test bus instance for testing, and returns a linked
// controller to manipulate the instance.
func SetupTestBus() *TestBus {
	t := newTestBus()
	testBusInstance.Store(t)
	return t
}

// To facilitate testing, dbusConn is an interface that encompasses the required
// subset of *dbus.Conn.
type dbusConn interface {
	BusObject() dbus.BusObject
	Close() error
	Object(string, dbus.ObjectPath) dbus.BusObject
	RemoveSignal(chan<- *dbus.Signal)
	Signal(chan<- *dbus.Signal)
}

const (
	bus   string = "org.freedesktop.DBus"
	props string = "org.freedesktop.DBus.Properties"

	busPath dbus.ObjectPath = "/org/freedesktop/DBus"
)

var (
	listNames        = dbusName{bus, "ListNames"}
	getNameOwner     = dbusName{bus, "GetNameOwner"}
	nameOwnerChanged = dbusName{bus, "NameOwnerChanged"}

	propsChanged = dbusName{props, "PropertiesChanged"}
)

// dbusName represents a DBus name, specifying an interface and member pair.
type dbusName struct {
	iface  string
	member string
}

func (d dbusName) call(c dbusConn, args ...interface{}) *dbus.Call {
	return c.BusObject().Call(d.String(), 0, args...)
}

func (d dbusName) addMatch(c dbusConn, args ...dbus.MatchOption) *dbus.Call {
	return c.BusObject().AddMatchSignal(d.iface, d.member, args...)
}

func (d dbusName) removeMatch(c dbusConn, args ...dbus.MatchOption) *dbus.Call {
	return c.BusObject().RemoveMatchSignal(d.iface, d.member, args...)
}

func (d dbusName) String() string {
	return expand(d.iface, d.member)
}

func connect(bus *dbus.Conn, err error) dbusConn {
	if err == nil {
		err = bus.Auth(nil)
	}
	if err == nil {
		err = bus.Hello()
	}
	if err != nil {
		panic("Could not connect to dbus: " + err.Error())
	}
	return bus
}

func shorten(iface, name string) string {
	if !strings.HasPrefix(name, iface+".") {
		return name
	}
	short := strings.TrimPrefix(name, iface+".")
	if strings.IndexRune(short, '.') < 0 {
		return short
	}
	return "." + short
}

func expand(iface, name string) string {
	switch strings.IndexRune(name, '.') {
	case 0:
		return iface + name
	case -1:
		return iface + "." + name
	default:
		return name
	}
}

func makeDbusName(str string) dbusName {
	lastDot := strings.LastIndexByte(str, byte('.'))
	if lastDot == -1 {
		return dbusName{"", str}
	}
	return dbusName{str[:lastDot], str[lastDot+1:]}
}
