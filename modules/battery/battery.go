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
	"bufio"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
)

// Info represents the current battery information.
type Info struct {
	// Capacity in *percents*, from 0 to 100.
	Capacity int
	// Energy when the battery is full, in Wh.
	EnergyFull float64
	// Max Energy the battery can store, in Wh.
	EnergyMax float64
	// Energy currently stored in the battery, in Wh.
	EnergyNow float64
	// Power currently being drawn from the battery, in W.
	Power float64
	// Current voltage of the batter, in V.
	Voltage float64
	// Status of the battery, e.g. "Charging", "Full", "Disconnected".
	Status string
	// Technology of the battery, e.g. "Li-Ion", "Li-Poly", "Ni-MH".
	Technology string
}

// Remaining returns the fraction of battery capacity remaining.
func (i Info) Remaining() float64 {
	if math.Nextafter(i.EnergyFull, 0) == 0 {
		return 0
	}
	return float64(i.EnergyNow) / float64(i.EnergyFull)
}

// RemainingPct returns the percentage of battery capacity remaining.
func (i Info) RemainingPct() int {
	return int(i.Remaining() * 100)
}

// RemainingTime returns the best guess for remaining time.
// This is based on the current power draw and remaining capacity.
// TODO: Moving average?
func (i Info) RemainingTime() time.Duration {
	// Battery does not report current draw,
	// cannot estimate remaining time.
	if math.Nextafter(i.Power, 0) == 0 {
		return time.Duration(0)
	}
	// ACPI spec says this must be in hours.
	hours := float64(i.EnergyNow) / float64(i.Power)
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

	// RefreshInterval configures the polling frequency for battery info.
	RefreshInterval(time.Duration) Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Info) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OutputColor configures a module to change the colour of its output based on a
	// user-defined function. This allows you to set up color thresholds, or even
	// blend between two colours based on the current battery state.
	OutputColor(func(Info) bar.Color) Module

	// UrgentWhen configures a module to mark its output as urgent based on a
	// user-defined function.
	UrgentWhen(func(Info) bool) Module
}

type module struct {
	*base.Base
	batteryName string
	scheduler   scheduler.Scheduler
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
	m.scheduler = scheduler.Do(m.Update).Every(3 * time.Second)
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

func (m *module) OutputFunc(outputFunc func(Info) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

func (m *module) RefreshInterval(interval time.Duration) Module {
	m.scheduler.Every(interval)
	return m
}

func (m *module) OutputColor(colorFunc func(Info) bar.Color) Module {
	m.colorFunc = colorFunc
	m.Update()
	return m
}

func (m *module) UrgentWhen(urgentFunc func(Info) bool) Module {
	m.urgentFunc = urgentFunc
	m.Update()
	return m
}

func (m *module) update() {
	info := batteryInfo(m.batteryName)
	out := m.outputFunc(info)
	if m.urgentFunc != nil {
		out.Urgent(m.urgentFunc(info))
	}
	if m.colorFunc != nil {
		out.Color(m.colorFunc(info))
	}
	m.Output(out)
}

// electricValue represents a value that is either watts or amperes.
// ACPI permits several of the properties to be in either unit, so to
// simplify reading such values, this type can represent either unit
// and convert as needed.
type electricValue struct {
	value   float64
	isWatts bool
}

func (e electricValue) toWatts(voltage float64) float64 {
	if e.isWatts {
		return e.value
	}
	return e.value * voltage
}

// uwatts constructs an electricValue from a string in micro-watts.
func uwatts(value string) electricValue {
	return electricValue{fromMicroStr(value), true}
}

// uamps constructs an electricValue from a string in micro-amps.
func uamps(value string) electricValue {
	return electricValue{fromMicroStr(value), false}
}

func fromMicroStr(str string) float64 {
	uValue, _ := strconv.Atoi(str)
	return float64(uValue) / math.Pow(10, 6 /* micros */)
}

var fs = afero.NewOsFs()

func batteryPath(name string) string {
	return fmt.Sprintf("/sys/class/power_supply/%s/uevent", name)
}

func batteryInfo(name string) Info {
	f, err := fs.Open(batteryPath(name))
	if err != nil {
		return Info{Status: "Disconnected"}
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)

	info := Info{Status: "Unknown"}
	var energyNow, powerNow, energyFull, energyMax electricValue
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if !strings.Contains(line, "=") {
			continue
		}
		split := strings.Split(line, "=")
		if len(split) != 2 {
			continue
		}
		key := strings.TrimPrefix(split[0], "POWER_SUPPLY_")
		value := split[1]
		switch key {
		case "CHARGE_NOW":
			energyNow = uamps(value)
		case "ENERGY_NOW":
			energyNow = uwatts(value)
		case "CHARGE_FULL":
			energyFull = uamps(value)
		case "ENERGY_FULL":
			energyFull = uwatts(value)
		case "CHARGE_FULL_DESIGN":
			energyMax = uamps(value)
		case "ENERGY_FULL_DESIGN":
			energyMax = uwatts(value)
		case "CURRENT_NOW":
			powerNow = uamps(value)
		case "POWER_NOW":
			powerNow = uwatts(value)
		case "VOLTAGE_NOW":
			info.Voltage = fromMicroStr(value)
		case "STATUS":
			info.Status = value
		case "TECHNOLOGY":
			info.Technology = value
		case "CAPACITY":
			info.Capacity, _ = strconv.Atoi(value)
		}
	}
	info.EnergyNow = energyNow.toWatts(info.Voltage)
	info.EnergyMax = energyMax.toWatts(info.Voltage)
	info.EnergyFull = energyFull.toWatts(info.Voltage)
	info.Power = powerNow.toWatts(info.Voltage)
	return info
}
