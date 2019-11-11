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

// Package bluetooth provides modules for watching the status of Bluetooth adapters and devices.
package bluetooth // import "barista.run/modules/bluetooth"

import (
	"strings"

	godbus "github.com/godbus/dbus/v5"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/base/watchers/dbus"
)

// DeviceModule represents a Bluetooth devices bar module.
type DeviceModule struct {
	path       string
	outputFunc value.Value
}

// DeviceInfo represents Bluetooth device information.
type DeviceInfo struct {
	Name      string
	Alias     string
	Address   string
	Adapter   string
	Battery   int
	Paired    bool
	Connected bool
	Trusted   bool
	Blocked   bool
}

// Device constructs a bluetooth device module instance for the given adapter and MAC address.
func Device(adapter, mac string) *DeviceModule {
	macPath := strings.Replace(strings.ToUpper(mac), ":", "_", -1)
	return &DeviceModule{path: "/org/bluez/" + adapter + "/dev_" + macPath}
}

// Output configures a module to display the output of a user-defined function.
func (m *DeviceModule) Output(outputFunc func(DeviceInfo) bar.Output) *DeviceModule {
	m.outputFunc.Set(outputFunc)
	return m
}

// Stream starts the module.
func (m *DeviceModule) Stream(sink bar.Sink) {
	w := dbus.WatchProperties(
		busType,
		"org.bluez",
		m.path,
		"org.bluez.Device1",
	).
		Add("Name", "Alias", "Address", "Adapter", "Paired", "Connected", "Trusted", "Blocked")
	defer w.Unsubscribe()

	batt := dbus.WatchProperties(
		busType,
		"org.bluez",
		m.path,
		"org.bluez.Battery1",
	).Add("Percentage")
	defer batt.Unsubscribe()

	outputFunc := m.outputFunc.Get().(func(DeviceInfo) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()

	info := getDeviceInfo(w, batt)
	for {
		sink.Output(outputFunc(info))
		select {
		case <-w.Updates:
			info = getDeviceInfo(w, batt)
		case <-batt.Updates:
			info = getDeviceInfo(w, batt)
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(DeviceInfo) bar.Output)
		}
	}
}

func getDeviceInfo(w, batt *dbus.PropertiesWatcher) DeviceInfo {
	i := DeviceInfo{}
	props := w.Get()

	i.Name, _ = props["Name"].(string)
	i.Alias, _ = props["Alias"].(string)
	i.Address, _ = props["Address"].(string)

	if adapter, ok := props["Adapter"].(godbus.ObjectPath); ok {
		i.Adapter = string(adapter)
	}

	i.Paired, _ = props["Paired"].(bool)
	i.Connected, _ = props["Connected"].(bool)
	i.Trusted, _ = props["Trusted"].(bool)
	i.Blocked, _ = props["Blocked"].(bool)
	if battery, ok := batt.Get()["Percentage"].(byte); ok {
		i.Battery = int(battery)
	}
	return i
}
