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

// Package systemd provides modules for watching the status of a systemd unit.
package systemd

import (
	"strings"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/value"
	"github.com/soumya92/barista/base/watchers/dbus"
	"github.com/soumya92/barista/base/watchers/localtz"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"

	systemdbus "github.com/coreos/go-systemd/v22/dbus"
)

// State represents possible states of a systemd unit.
type State string

const (
	// StateUnknown indicates an unknown unit state.
	StateUnknown = State("")
	// StateActive indicates that unit is active.
	StateActive = State("active")
	// StateReloading indicates that the unit is active and currently reloading
	// its configuration.
	StateReloading = State("reloading")
	// StateInactive indicates that it is inactive and the previous run was
	// successful or no previous run has taken place yet.
	StateInactive = State("inactive")
	// StateFailed indicates that it is inactive and the previous run was not
	// successful.
	StateFailed = State("failed")
	// StateActivating indicates that the unit has previously been inactive but
	// is currently in the process of entering an active state.
	StateActivating = State("activating")
	// StateDeactivating indicates that the unit is currently in the process of
	// deactivation.
	StateDeactivating = State("deactivating")
)

// UnitInfo includes common information present in both services and timers.
type UnitInfo struct {
	ID          string
	Description string
	State       State
	SubState    string
	Since       time.Time
	// DBus 'call' method to control the unit.
	call func(string, ...interface{}) ([]interface{}, error)
}

// Start enqueues a start job, and possibly dependant jobs.
func (u UnitInfo) Start() {
	u.call("Start", "fail")
}

// Stop stops the specified unit rather than starting it.
func (u UnitInfo) Stop() {
	u.call("Stop", "fail")
}

// Restart restarts a unit. If a service is restarted that isn't running it will
// be started.
func (u UnitInfo) Restart() {
	u.call("Restart", "fail")
}

// Reload reloads a unit. Reloading is done only if the unit is already running
// and fails otherwise.
func (u UnitInfo) Reload() {
	u.call("Reload", "fail")
}

func watchUnit(name string, busType ...dbus.BusType) *dbus.PropertiesWatcher {
	escapedName := systemdbus.PathBusEscape(name)
	unitPath := "/org/freedesktop/systemd1/unit/" + escapedName
	return dbus.WatchProperties(getBusType(busType...),
		"org.freedesktop.systemd1", unitPath, "org.freedesktop.systemd1.Unit").
		Add("ActiveState", "SubState", "Id", "Description").
		FetchOnSignal("StateChangeTimestamp")
}

// ServiceInfo represents the state of a systemd service.
type ServiceInfo struct {
	UnitInfo
	Type    string
	ExecPID uint32
	MainPID uint32
}

// ServiceModule watches a systemd service and updates on status change
type ServiceModule struct {
	name       string
	busType    dbus.BusType
	outputFunc value.Value
}

// UserService create a module that watches the status of a systemd user service.
func UserService(name string) *ServiceModule {
	return service(name, dbus.Session)
}

// Service creates a module that watches the status of a systemd service.
func Service(name string) *ServiceModule {
	return service(name, dbus.System)
}

func service(name string, dbusType dbus.BusType) *ServiceModule {
	s := &ServiceModule{name: name, busType: dbusType}
	s.Output(func(i ServiceInfo) bar.Output {
		if i.Since.IsZero() {
			return outputs.Textf("%s (%s)", i.State, i.SubState)
		}
		since := i.Since.Format("15:04")
		if timing.Now().Add(-24 * time.Hour).After(i.Since) {
			since = i.Since.Format("Jan 2")
		}
		return outputs.Textf("%s (%s) since %s", i.State, i.SubState, since)
	})
	return s
}

// Output configures a module to display the output of a user-defined function.
func (s *ServiceModule) Output(outputFunc func(ServiceInfo) bar.Output) *ServiceModule {
	s.outputFunc.Set(outputFunc)
	return s
}

const serviceIface = "org.freedesktop.systemd1.Service"

// Stream starts the module.
func (s *ServiceModule) Stream(sink bar.Sink) {
	w := watchUnit(s.name+".service", s.busType)
	defer w.Unsubscribe()

	w.FetchOnSignal(
		serviceIface+".Type",
		serviceIface+".MainPID",
		serviceIface+".ExecMainPID",
	)

	outputFunc := s.outputFunc.Get().(func(ServiceInfo) bar.Output)
	nextOutputFunc, done := s.outputFunc.Subscribe()
	defer done()

	info := getServiceInfo(w)
	for {
		sink.Output(outputFunc(info))
		select {
		case <-w.Updates:
			info = getServiceInfo(w)
		case <-nextOutputFunc:
			outputFunc = s.outputFunc.Get().(func(ServiceInfo) bar.Output)
		}
	}
}

const usecInSec = 1000 * 1000

func timeFromUsec(usecValue interface{}) time.Time {
	usec, _ := usecValue.(uint64)
	if usec == 0 {
		return time.Time{}
	}
	sec := int64(usec / usecInSec)
	usecOnly := int64(usec % usecInSec)
	return time.Unix(sec, usecOnly*1000 /* nsec */).In(localtz.Get())
}

func getUnitInfo(w *dbus.PropertiesWatcher) (UnitInfo, map[string]interface{}) {
	u := UnitInfo{call: w.Call}
	props := w.Get()
	if s, ok := props["ActiveState"].(string); ok {
		u.State = State(s)
	}
	u.ID, _ = props["Id"].(string)
	u.Description, _ = props["Description"].(string)
	u.SubState, _ = props["SubState"].(string)
	u.Since = timeFromUsec(props["StateChangeTimestamp"])
	return u, props
}

func getServiceInfo(w *dbus.PropertiesWatcher) ServiceInfo {
	i := ServiceInfo{}
	var props map[string]interface{}
	i.UnitInfo, props = getUnitInfo(w)
	i.ID = strings.TrimSuffix(i.ID, ".service")
	if mPid, ok := props[serviceIface+".MainPID"].(uint32); ok {
		i.MainPID = mPid
	}
	if ePid, ok := props[serviceIface+".ExecMainPID"].(uint32); ok {
		i.ExecPID = ePid
	}
	i.Type, _ = props[serviceIface+".Type"].(string)
	return i
}

func getBusType(busType ...dbus.BusType) dbus.BusType {
	if len(busType) == 0 {
		return dbus.System
	}

	return busType[0]
}

// TimerInfo represents the state of a systemd timer.
type TimerInfo struct {
	UnitInfo
	Unit        string
	LastTrigger time.Time
	NextTrigger time.Time
}

// TimerModule watches a systemd timer and updates on status change
type TimerModule struct {
	name       string
	busType    dbus.BusType
	outputFunc value.Value
}

// UserTimer creates a module that watches the status of a systemd user timer.
func UserTimer(name string) *TimerModule {
	return timer(name, dbus.Session)
}

// Timer creates a module that watches the status of a systemd timer.
func Timer(name string) *TimerModule {
	return timer(name, dbus.System)
}

func timer(name string, dbusType dbus.BusType) *TimerModule {
	t := &TimerModule{name: name, busType: dbusType}
	t.Output(func(i TimerInfo) bar.Output {
		last := "never"
		if !i.LastTrigger.IsZero() {
			last = i.LastTrigger.Format("Jan 2, 15:04")
		}
		next := "never"
		if !i.NextTrigger.IsZero() {
			next = i.NextTrigger.Format("Jan 2, 15:04")
		}
		return outputs.Textf("%s@%s (last:%s)", i.Unit, next, last)
	})
	return t
}

// Output configures a module to display the output of a user-defined function.
func (t *TimerModule) Output(outputFunc func(TimerInfo) bar.Output) *TimerModule {
	t.outputFunc.Set(outputFunc)
	return t
}

const timerIface = "org.freedesktop.systemd1.Timer"

// Stream starts the module.
func (t *TimerModule) Stream(sink bar.Sink) {
	w := watchUnit(t.name+".timer", t.busType)
	defer w.Unsubscribe()

	w.FetchOnSignal(
		timerIface+".Unit",
		timerIface+".LastTriggerUSec",
		timerIface+".NextElapseUSecRealtime",
	)

	outputFunc := t.outputFunc.Get().(func(TimerInfo) bar.Output)
	nextOutputFunc, done := t.outputFunc.Subscribe()
	defer done()

	info := getTimerInfo(w)
	for {
		sink.Output(outputFunc(info))
		select {
		case <-w.Updates:
			info = getTimerInfo(w)
		case <-nextOutputFunc:
			outputFunc = t.outputFunc.Get().(func(TimerInfo) bar.Output)
		}
	}
}

func getTimerInfo(w *dbus.PropertiesWatcher) TimerInfo {
	i := TimerInfo{}
	var props map[string]interface{}
	i.UnitInfo, props = getUnitInfo(w)
	i.ID = strings.TrimSuffix(i.ID, ".timer")
	i.LastTrigger = timeFromUsec(props[timerIface+".LastTriggerUSec"])
	i.NextTrigger = timeFromUsec(props[timerIface+".NextElapseUSecRealtime"])
	i.Unit, _ = props[timerIface+".Unit"].(string)
	return i
}
