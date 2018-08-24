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
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
)

// Status represents a normalised battery status.
type Status string

const (
	// Disconnected represents a named battery that was not found.
	Disconnected Status = "Disconnected"
	// Charging represents a battery that is actively being charged.
	Charging Status = "Charging"
	// Discharging represents a battery that is actively being discharged.
	Discharging Status = "Discharging"
	// Full represents a battery that is plugged in and at capacity.
	Full Status = "Full"
	// NotCharging represents a battery that is plugged in,
	// not full, but not charging.
	NotCharging Status = "Not charging"
	// Unknown is used to catch all other statuses.
	Unknown Status = ""
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
	Status Status
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
func (i Info) RemainingTime() time.Duration {
	// Battery does not report current draw,
	// cannot estimate remaining time.
	if math.Nextafter(i.Power, 0) == 0 {
		return 0
	}
	// According to ACPI spec, these calculations will return hours.
	hours := 0.0
	switch i.Status {
	case Charging:
		hours = (i.EnergyFull - i.EnergyNow) / i.Power
	case Discharging:
		hours = i.EnergyNow / i.Power
	}
	return time.Duration(int(hours*3600)) * time.Second
}

// Discharging returns true if the battery is being discharged.
func (i Info) Discharging() bool {
	return i.Status != Charging && i.Status != Full
}

// PluggedIn returns true if the laptop is plugged in.
func (i Info) PluggedIn() bool {
	return i.Status == Charging || i.Status == Full || i.Status == NotCharging
}

// SignedPower returns a positive power value when the battery
// is being charged, and a negative power value when discharged.
func (i Info) SignedPower() float64 {
	if i.Discharging() {
		return -i.Power
	}
	return i.Power
}

// Module represents a battery bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module struct {
	updateFunc func() Info
	scheduler  timing.Scheduler
	format     base.Value
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

func newModule(updateFunc func() Info) *Module {
	m := &Module{
		updateFunc: updateFunc,
		scheduler:  timing.NewScheduler(),
	}
	l.Register(m, "scheduler", "format")
	m.format.Set(format{})
	m.RefreshInterval(3 * time.Second)
	// Construct a simple template that's just the available battery percent.
	m.Template(`BATT {{.RemainingPct}}%`)
	return m
}

// Named constructs an instance of the battery module for the given battery name.
func Named(name string) *Module {
	m := newModule(func() Info { return batteryInfo(name) })
	l.Label(m, name)
	return m
}

// All constructs a battery module that aggregates all detected batteries.
func All() *Module {
	return newModule(allBatteriesInfo)
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	c := m.getFormat()
	c.outputFunc = outputFunc
	m.format.Set(c)
	return m
}

// Template configures a module to display the output of a template.
func (m *Module) Template(template string) *Module {
	base.Template(template, m.Output)
	return m
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
func (m *Module) Stream(s bar.Sink) {
	info := m.updateFunc()
	format := m.getFormat()
	for {
		s.Output(format.output(info))
		select {
		case <-m.scheduler.Tick():
			info = m.updateFunc()
		case <-m.format.Next():
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

func fromStatusStr(str string) Status {
	switch str {
	case string(Full):
		return Full
	case string(Charging):
		return Charging
	case string(Discharging):
		return Discharging
	case string(Disconnected):
		return Disconnected
	case string(NotCharging):
		return NotCharging
	default:
		return Unknown
	}
}

var fs = afero.NewOsFs()

func batteryInfo(name string) Info {
	batteryPath := fmt.Sprintf("/sys/class/power_supply/%s/uevent", name)
	l.Fine("Reading from %s", batteryPath)
	f, err := fs.Open(batteryPath)
	if err != nil {
		l.Log("Failed to read stats for %s: %s", name, err)
		return Info{Status: Disconnected}
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)

	var info Info
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
			info.Status = fromStatusStr(value)
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

func allBatteriesInfo() Info {
	dir, err := fs.Open("/sys/class/power_supply")
	if err != nil {
		l.Log("No batteries: %s", err)
		return Info{Status: Disconnected}
	}
	batts, err := dir.Readdirnames(-1)
	if err != nil {
		l.Log("Failed to list batteries: %s", err)
		return Info{Status: Unknown}
	}
	var infos []Info
	for _, batt := range batts {
		if !strings.HasPrefix(batt, "BAT") {
			continue
		}
		infos = append(infos, batteryInfo(batt))
	}
	if len(infos) == 0 {
		return Info{Status: Disconnected}
	}
	var allInfo Info
	var techs []string
	var voltEnergySum float64
	for _, info := range infos {
		allInfo.EnergyFull += info.EnergyFull
		allInfo.EnergyMax += info.EnergyMax
		allInfo.EnergyNow += info.EnergyNow
		if info.Technology != "" {
			techs = append(techs, info.Technology)
		}
		voltEnergySum += info.Voltage * info.EnergyNow
		signedPower := allInfo.SignedPower() + info.SignedPower()
		allInfo.Power = math.Abs(signedPower)

		switch allInfo.Status {
		case Charging:
			if signedPower < 0 {
				allInfo.Status = Discharging
			}
		case Discharging:
			if signedPower > 0 {
				allInfo.Status = Charging
			}
		default:
			if info.Status != Unknown {
				allInfo.Status = info.Status
			}
		}
	}
	// No meaningful voltage aggregator, so just average it by the energy
	// stored at each voltage. (e.g. 10Wh @ 12V, 5Wh @ 9V = ~11V).
	allInfo.Voltage = voltEnergySum / allInfo.EnergyNow
	allInfo.Capacity = int(allInfo.EnergyNow * 100.0 / allInfo.EnergyFull)
	allInfo.Technology = strings.Join(techs, ",")
	return allInfo
}
