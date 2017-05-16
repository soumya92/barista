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

// Module represents a battery bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module interface {
	base.WithClickHandler
	RefreshInterval(time.Duration) Module
	OutputFunc(func(Info) bar.Output) Module
	OutputTemplate(func(interface{}) bar.Output) Module
	OutputColor(func(Info) bar.Color) Module
	UrgentWhen(func(Info) bool) Module
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *module) OutputFunc(outputFunc func(Info) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

// RefreshInterval configures the polling frequency for battery info.
func (m *module) RefreshInterval(interval time.Duration) Module {
	m.scheduler.Stop()
	m.scheduler = m.UpdateEvery(interval)
	return m
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current battery state.
func (m *module) OutputColor(colorFunc func(Info) bar.Color) Module {
	m.colorFunc = colorFunc
	m.Update()
	return m
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
func (m *module) UrgentWhen(urgentFunc func(Info) bool) Module {
	m.urgentFunc = urgentFunc
	m.Update()
	return m
}

type module struct {
	*base.Base
	batteryName string
	scheduler   base.Scheduler
	outputFunc  func(Info) bar.Output
	colorFunc   func(Info) bar.Color
	urgentFunc  func(Info) bool
}

// New constructs an instance of the battery module for the given battery name.
func New(name string) Module {
	m := &module{
		Base:        base.New(),
		batteryName: name,
	}
	// Default is to refresh every 3s, matching the behaviour of top.
	m.scheduler = m.UpdateEvery(3 * time.Second)
	// Construct a simple template that's just the available battery percent.
	m.OutputTemplate(outputs.TextTemplate(`BATT {{.RemainingPct}}%`))
	// Update battery stats whenever an update is requested.
	m.OnUpdate(m.update)
	return m
}

// Default constructs an instance of the battery module with BAT0.
func Default() Module {
	return New("BAT0")
}

func (m *module) update() {
	info, err := m.batteryInfo()
	if m.Error(err) {
		return
	}
	out := m.outputFunc(info)
	if m.urgentFunc != nil {
		out.Urgent(m.urgentFunc(info))
	}
	if m.colorFunc != nil {
		out.Color(m.colorFunc(info))
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
