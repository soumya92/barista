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

// Package bluetooth provides modules for watching the status of Bluetooth adapters.
package bluetooth // import "barista.run/modules/bluetooth"

import (
	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/base/watchers/dbus"
)

// Module represents a Bluetooth bar module.
type Module struct {
	adapter    string
	outputFunc value.Value
}

// AdapterInfo represents a Bluetooth adapters information.
type AdapterInfo struct {
	Name         string
	Alias        string
	Address      string
	Discoverable bool
	Pairable     bool
	Powered      bool
	Discovering  bool
}

// DefaultAdapter constructs an instance of the bluetooth module using the first adapter ("hci0").
func DefaultAdapter() *Module {
	return Adapter("hci0")
}

// Adapter constructs an instance of the bluetooth module with the provided device name (ex. "hci1").
func Adapter(name string) *Module {
	bt := &Module{adapter: name}
	return bt
}

// Output configures a module to display the output of a user-defined function.
func (bt *Module) Output(outputFunc func(AdapterInfo) bar.Output) *Module {
	bt.outputFunc.Set(outputFunc)
	return bt
}

// Stream starts the module.
func (bt *Module) Stream(sink bar.Sink) {
	w := dbus.WatchProperties(
		dbus.System,
		"org.bluez",
		"/org/bluez/"+bt.adapter,
		"org.bluez.Adapter1",
	).
		Add("Name", "Alias", "Address", "Discoverable", "Pairable", "Powered", "Discovering")
	defer w.Unsubscribe()

	outputFunc := bt.outputFunc.Get().(func(AdapterInfo) bar.Output)
	nextOutputFunc, done := bt.outputFunc.Subscribe()
	defer done()

	info := getAdapterInfo(w)
	for {
		sink.Output(outputFunc(info))
		select {
		case <-w.Updates:
			info = getAdapterInfo(w)
		case <-nextOutputFunc:
			outputFunc = bt.outputFunc.Get().(func(AdapterInfo) bar.Output)
		}
	}
}

func getAdapterInfo(w *dbus.PropertiesWatcher) AdapterInfo {
	i := AdapterInfo{}
	props := w.Get()

	if name, ok := props["Name"].(string); ok {
		i.Name = name
	}
	i.Alias, _ = props["Alias"].(string)
	i.Address, _ = props["Address"].(string)
	i.Discoverable, _ = props["Discoverable"].(bool)
	i.Pairable, _ = props["Pairable"].(bool)
	i.Powered, _ = props["Powered"].(bool)
	i.Discovering, _ = props["Discovering"].(bool)

	return i
}
