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

package bluetooth

import (
	"strings"
	"testing"

	godbus "github.com/godbus/dbus"

	"barista.run/bar"
	"barista.run/base/watchers/dbus"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
)

func TestDevice(t *testing.T) {
	testBar.New(t)

	adapterName := "hci0"
	deviceMac := "00:00:00:00:25:9F"
	device, battery := setupTestDevice(adapterName, deviceMac)
	device.SetProperties(map[string]interface{}{
		"Name":      "foo",
		"Alias":     "foo alias",
		"Address":   deviceMac,
		"Paired":    true,
		"Connected": true,
		"Trusted":   true,
		"Blocked":   false,
	}, dbus.SignalTypeNone)
	battery.SetProperty("Percentage", byte(40), dbus.SignalTypeNone)

	devModule := Device(adapterName, deviceMac)
	devModule.Output(func(i DeviceInfo) bar.Output {
		paired := "NO"
		if i.Paired {
			paired = "YES"
		}
		return outputs.Textf("%s: %s (%d%%)", i.Name, paired, i.Battery)
	})
	testBar.Run(devModule)

	testBar.NextOutput().AssertText([]string{"foo: YES (40%)"})

	battery.SetProperty("Percentage", byte(30), dbus.SignalTypeChanged)
	testBar.NextOutput().AssertText([]string{"foo: YES (30%)"})
}

func TestDeviceDisconnect(t *testing.T) {
	testBar.New(t)

	adapterName := "hci0"
	deviceMac := "00:00:00:00:25:9F"
	device, _ := setupTestDevice(adapterName, deviceMac)
	device.SetProperties(map[string]interface{}{
		"Name":      "foo",
		"Alias":     "foo alias",
		"Address":   deviceMac,
		"Paired":    true,
		"Connected": true,
		"Trusted":   true,
		"Blocked":   false,
	}, dbus.SignalTypeNone)

	devModule := Device(adapterName, deviceMac)
	devModule.Output(func(i DeviceInfo) bar.Output {
		connected := "NO"
		if i.Connected {
			connected = "YES"
		}
		return outputs.Textf("%s", connected)
	})
	testBar.Run(devModule)

	testBar.NextOutput().AssertText([]string{"YES"})

	device.SetProperty("Connected", false, dbus.SignalTypeChanged)
	testBar.NextOutput().AssertText([]string{"NO"})
}

func setupTestDevice(adapterName, deviceMac string) (device, battery *dbus.TestBusObject) {
	bus := dbus.SetupTestBus()
	bluez := bus.RegisterService("org.bluez")

	devicePath := "dev_" + strings.Replace(deviceMac, ":", "_", -1)
	deviceObjPath := godbus.ObjectPath("/org/bluez/" + adapterName + "/" + devicePath)

	device = bluez.Object(deviceObjPath, "org.bluez.Device1")
	battery = bluez.Object(deviceObjPath, "org.bluez.Battery1")
	return // device, battery
}
