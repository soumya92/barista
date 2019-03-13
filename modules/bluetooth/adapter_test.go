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
	"testing"

	godbus "github.com/godbus/dbus"

	"barista.run/bar"
	"barista.run/base/watchers/dbus"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
)

func init() {
	busType = dbus.Test
}

func TestAdapter(t *testing.T) {
	testBar.New(t)

	adapterName := "hci0"
	adapter := setupTestAdapter(adapterName)
	adapter.SetProperties(map[string]interface{}{
		"Name":         "foo",
		"Alias":        "foo alias",
		"Address":      "28:C2:DD:8B:73:8C",
		"Discoverable": false,
		"Pairable":     true,
		"Powered":      true,
		"Discovering":  false,
	}, dbus.SignalTypeNone)

	btModule := Adapter(adapterName)
	btModule.Output(func(i AdapterInfo) bar.Output {
		state := "OFF"
		if i.Powered {
			state = "ON"
		}
		return outputs.Textf("%s: %s", i.Name, state)
	})
	testBar.Run(btModule)

	testBar.LatestOutput().AssertText([]string{
		"foo: ON",
	})
}

func TestAdapterDisconnect(t *testing.T) {
	testBar.New(t)

	adapterName := "hci0"
	adapter := setupTestAdapter(adapterName)
	adapter.SetProperties(map[string]interface{}{
		"Name":         "foo",
		"Alias":        "foo alias",
		"Address":      "28:C2:DD:8B:73:8C",
		"Discoverable": false,
		"Pairable":     true,
		"Powered":      true,
		"Discovering":  false,
	}, dbus.SignalTypeNone)

	btModule := DefaultAdapter()
	btModule.Output(func(i AdapterInfo) bar.Output {
		state := "OFF"
		if i.Powered {
			state = "ON"
		}
		return outputs.Textf("%s", state)
	})
	testBar.Run(btModule)

	testBar.LatestOutput().AssertText([]string{
		"ON",
	})

	adapter.SetPropertyForTest("Powered", false, dbus.SignalTypeChanged)

	testBar.LatestOutput().AssertText([]string{
		"OFF",
	})
}

func setupTestAdapter(adapterName string) *dbus.TestBusObject {
	bus := dbus.SetupTestBus()
	bluez := bus.RegisterService("org.bluez")

	adapterObjPath := godbus.ObjectPath("/org/bluez/" + adapterName)
	adapter := bluez.Object(adapterObjPath, "org.bluez.Adapter1")

	return adapter
}
