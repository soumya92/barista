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

// Package cputemp implements an i3bar module that shows the CPU temperature.
package cputemp

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/modules/base"
)

// Temperature represents the current CPU temperature.
type Temperature float64

// C returns the temperature in degrees celcius.
func (t Temperature) C() int {
	return int(t)
}

// K returns the temperature in kelvin.
func (t Temperature) K() int {
	return int(float64(t) + 273.15)
}

// F returns the temperature in degrees fahrenheit.
func (t Temperature) F() int {
	return int(float64(t)*1.8 + 32)
}

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(Temperature) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(t Temperature) *bar.Output {
		return template(t)
	})
}

// ThermalZone sets the thermal zone to read cpu temperature from.
// The file /sys/class/thermal/<zone>/temp should return cpu temp in 1/1000 deg C.
type ThermalZone string

func (t ThermalZone) apply(m *module) {
	m.thermalZone = string(t)
}

// RefreshInterval configures the polling frequency for cpu temperatures.
// Note: updates might still be less frequent if the temperature does not change.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current temperature.
type OutputColor func(Temperature) bar.Color

func (o OutputColor) apply(m *module) {
	m.colorFunc = o
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
type UrgentWhen func(Temperature) bool

func (u UrgentWhen) apply(m *module) {
	m.urgentFunc = u
}

type module struct {
	*base.Base
	thermalZone     string
	refreshInterval time.Duration
	outputFunc      func(Temperature) *bar.Output
	colorFunc       func(Temperature) bar.Color
	urgentFunc      func(Temperature) bool
	// Store last cpu temp to skip updates when unchanged.
	lastTempMilliC int
}

// New constructs an instance of the cputemp module with the provided configuration.
func New(config ...Config) base.Module {
	m := &module{
		Base: base.New(),
		// Default thermal zone for goobuntu. Override using ThermalFile(...)
		thermalZone: "thermal_zone0",
		// Default is to refresh every 3s, matching the behaviour of top.
		refreshInterval: 3 * time.Second,
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just the temperature in deg C.
		defTpl := outputs.TextTemplate(`{{.C}}℃`)
		OutputTemplate(defTpl).apply(m)
	}
	// Worker goroutine to update load average at a fixed interval.
	m.SetWorker(m.poll)
	return m
}

func (m *module) poll() error {
	// Ideally fsnotify would work for sensors as well, but it doesn't, so we'll
	// compromise by polling here but only updating the bar when the temperature
	// actually changes.
	thermalFile := fmt.Sprintf("/sys/class/thermal/%s/temp", m.thermalZone)
	for {
		if err := m.maybeUpdate(thermalFile); err != nil {
			return err
		}
		time.Sleep(m.refreshInterval)
	}
}

func (m *module) maybeUpdate(thermalFile string) error {
	bytes, err := ioutil.ReadFile(thermalFile)
	if err != nil {
		return err
	}
	value := strings.TrimSpace(string(bytes))
	milliC, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	if milliC == m.lastTempMilliC {
		return nil
	}
	temp := Temperature(float64(milliC) / 1000.0)
	out := m.outputFunc(temp)
	if m.urgentFunc != nil {
		out.Urgent = m.urgentFunc(temp)
	}
	if m.colorFunc != nil {
		out.Color = m.colorFunc(temp)
	}
	m.Output(out)
	m.lastTempMilliC = milliC
	return nil
}
