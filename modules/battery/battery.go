// Copyright 2017 Google Inc.
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

// Package battery provides a battery status i3bar module.
package battery

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
	"github.com/soumya92/barista/outputs"
)

// Info represents the current battery information.
type Info struct {
	Capacity   int
	EnergyFull int
	EnergyMax  int
	Energy     int
	Power      int
	VoltageMin int
	Voltage    int
	Status     string
	Technology string
}

// Remaining returns the fraction of battery capacity remaining.
func (i Info) Remaining() float64 {
	return float64(i.Energy) / float64(i.EnergyMax)
}

// RemainingPct returns the percentage of battery capacity remaining.
func (i Info) RemainingPct() int {
	return int(i.Remaining() * 100)
}

// RemainingTime returns the best guess for remaining time.
// This is based on the current power draw and remaining capacity.
// TODO: Moving average?
func (i Info) RemainingTime() time.Duration {
	// ACPI spec says this must be in hours.
	hours := float64(i.Energy) / float64(i.Power)
	return time.Duration(int(hours*3600)) * time.Second
}

// PluggedIn returns true if the laptop is plugged in.
func (i Info) PluggedIn() bool {
	return i.Status == "Charging" || i.Status == "Full"
}

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(Info) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(i Info) *bar.Output {
		return template(i)
	})
}

// Name sets the name of the battery, which controls the directory used
// for reading battery information from /sys/class/power_supply/.
type Name string

func (n Name) apply(m *module) {
	m.batteryName = string(n)
}

// RefreshInterval configures the polling frequency for battery info.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current battery state.
type OutputColor func(Info) bar.Color

func (o OutputColor) apply(m *module) {
	m.colorFunc = o
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
type UrgentWhen func(Info) bool

func (u UrgentWhen) apply(m *module) {
	m.urgentFunc = u
}

type module struct {
	*base.Base
	batteryName     string
	refreshInterval time.Duration
	outputFunc      func(Info) *bar.Output
	colorFunc       func(Info) bar.Color
	urgentFunc      func(Info) bool
}

// New constructs an instance of the cputemp module with the provided configuration.
func New(config ...Config) base.WithClickHandler {
	m := &module{
		Base: base.New(),
		// Default battery for goobuntu laptops. Override using BatteryName(...)
		batteryName: "BAT0",
		// Default is to refresh every 3s, matching the behaviour of top.
		refreshInterval: 3 * time.Second,
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just the available battery percent.
		defTpl := outputs.TextTemplate(`BATT {{.RemainingPct}}%`)
		OutputTemplate(defTpl).apply(m)
	}
	// Worker goroutine to update load average at a fixed interval.
	m.OnUpdate(m.update)
	m.UpdateEvery(m.refreshInterval)
	return m
}

func (m *module) update() {
	info, err := m.batteryInfo()
	if m.Error(err) {
		return
	}
	out := m.outputFunc(info)
	if m.urgentFunc != nil {
		out.Urgent = m.urgentFunc(info)
	}
	if m.colorFunc != nil {
		out.Color = m.colorFunc(info)
	}
	m.Output(out)
}

func (m *module) batteryInfo() (i Info, e error) {
	if i.Technology, e = m.readString("technology"); e != nil {
		return
	}
	if i.Status, e = m.readString("status"); e != nil {
		return
	}
	if i.Capacity, e = m.readInt("capacity"); e != nil {
		return
	}
	if i.EnergyFull, e = m.readInt("energy_full"); e != nil {
		return
	}
	if i.EnergyMax, e = m.readInt("energy_full_design"); e != nil {
		return
	}
	if i.Energy, e = m.readInt("energy_now"); e != nil {
		return
	}
	if i.Power, e = m.readInt("power_now"); e != nil {
		return
	}
	if i.VoltageMin, e = m.readInt("voltage_min_design"); e != nil {
		return
	}
	i.Voltage, e = m.readInt("voltage_now")
	return
}

func (m *module) readString(prop string) (string, error) {
	file := fmt.Sprintf("/sys/class/power_supply/%s/%s", m.batteryName, prop)
	bytes, err := ioutil.ReadFile(file)
	return strings.TrimSpace(string(bytes)), err
}

func (m *module) readInt(prop string) (int, error) {
	str, err := m.readString(prop)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(str)
}
