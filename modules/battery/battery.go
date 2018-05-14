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
	"image/color"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
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
	return i.EnergyNow / i.EnergyFull
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
	hours := i.EnergyNow / i.Power
	return time.Duration(int(hours*3600)) * time.Second
}

// PluggedIn returns true if the laptop is plugged in.
func (i Info) PluggedIn() bool {
	return i.Status == "Charging" || i.Status == "Full"
}

// Module represents a battery bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module struct {
	base.SimpleClickHandler
	batteryName string
	scheduler   timing.Scheduler
	format      base.Value
}

type format struct {
	outputFunc func(Info) bar.Output
	colorFunc  func(Info) color.Color
	urgentFunc func(Info) bool
}

func (f format) output(i Info) bar.Output {
	out := outputs.Group(f.outputFunc(i))
	if f.urgentFunc != nil {
		out.Urgent(f.urgentFunc(i))
	}
	if f.colorFunc != nil {
		out.Color(f.colorFunc(i))
	}
	return out
}

func (m *Module) getFormat() format {
	return m.format.Get().(format)
}

// New constructs an instance of the battery module for the given battery name.
func New(name string) *Module {
	m := &Module{
		batteryName: name,
		scheduler:   timing.NewScheduler().Every(3 * time.Second),
	}
	m.format.Set(format{})
	// Construct a simple template that's just the available battery percent.
	m.OutputTemplate(outputs.TextTemplate(`BATT {{.RemainingPct}}%`))
	return m
}

// Default constructs an instance of the battery module with BAT0.
func Default() *Module {
	return New("BAT0")
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *Module) OutputFunc(outputFunc func(Info) bar.Output) *Module {
	c := m.getFormat()
	c.outputFunc = outputFunc
	m.format.Set(c)
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *Module) OutputTemplate(template func(interface{}) bar.Output) *Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

// RefreshInterval configures the polling frequency for battery info.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current battery state.
func (m *Module) OutputColor(colorFunc func(Info) color.Color) *Module {
	c := m.getFormat()
	c.colorFunc = colorFunc
	m.format.Set(c)
	return m
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
func (m *Module) UrgentWhen(urgentFunc func(Info) bool) *Module {
	c := m.getFormat()
	c.urgentFunc = urgentFunc
	m.format.Set(c)
	return m
}

// Stream starts the module.
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *Module) worker(ch base.Channel) {
	info := batteryInfo(m.batteryName)
	format := m.getFormat()
	sFormat := m.format.Subscribe()
	for {
		ch.Output(format.output(info))
		select {
		case <-m.scheduler.Tick():
			info = batteryInfo(m.batteryName)
		case <-sFormat:
			format = m.getFormat()
		}
	}
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

func batteryInfo(name string) Info {
	batteryPath := fmt.Sprintf("/sys/class/power_supply/%s/uevent", name)
	f, err := fs.Open(batteryPath)
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
