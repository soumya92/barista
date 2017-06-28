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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
)

// Temperature represents the current CPU temperature.
type Temperature float64

// C returns the temperature in degrees celcius.
func (t Temperature) C() int {
	return int(math.Floor(float64(t) + .5))
}

// K returns the temperature in kelvin.
func (t Temperature) K() int {
	return int(math.Floor(float64(t) + 273.15 + .5))
}

// F returns the temperature in degrees fahrenheit.
func (t Temperature) F() int {
	return int(math.Floor(float64(t)*1.8 + 32 + .5))
}

// Module represents a cputemp bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module interface {
	base.WithClickHandler

	// RefreshInterval configures the polling frequency for cpu temperatures.
	// Note: updates might still be less frequent if the temperature does not change.
	RefreshInterval(time.Duration) Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Temperature) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OutputColor configures a module to change the colour of its output based on a
	// user-defined function. This allows you to set up color thresholds, or even
	// blend between two colours based on the current temperature.
	OutputColor(func(Temperature) bar.Color) Module

	// UrgentWhen configures a module to mark its output as urgent based on a
	// user-defined function.
	UrgentWhen(func(Temperature) bool) Module
}

type module struct {
	*base.Base
	thermalFile string
	outputFunc  func(Temperature) bar.Output
	colorFunc   func(Temperature) bar.Color
	urgentFunc  func(Temperature) bool
}

// Zone constructs an instance of the cputemp module for the specified zone.
// The file /sys/class/thermal/<zone>/temp should return cpu temp in 1/1000 deg C.
func Zone(thermalZone string) Module {
	m := &module{
		Base:        base.New(),
		thermalFile: fmt.Sprintf("/sys/class/thermal/%s/temp", thermalZone),
	}
	// Default is to refresh every 3s, matching the behaviour of top.
	m.RefreshInterval(3 * time.Second)
	// Default output template, if no template/function was specified.
	m.OutputTemplate(outputs.TextTemplate(`{{.C}}℃`))
	// Update temperature when asked.
	m.OnUpdate(m.update)
	return m
}

// DefaultZone constructs an instance of the cputemp module for the default zone.
func DefaultZone() Module {
	return Zone("thermal_zone0")
}

func (m *module) OutputFunc(outputFunc func(Temperature) bar.Output) Module {
	m.Lock()
	defer m.UnlockAndUpdate()
	m.outputFunc = outputFunc
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(t Temperature) bar.Output {
		return template(t)
	})
}

func (m *module) RefreshInterval(interval time.Duration) Module {
	m.Schedule().Every(interval)
	return m
}

func (m *module) OutputColor(colorFunc func(Temperature) bar.Color) Module {
	m.Lock()
	defer m.UnlockAndUpdate()
	m.colorFunc = colorFunc
	return m
}

func (m *module) UrgentWhen(urgentFunc func(Temperature) bool) Module {
	m.Lock()
	defer m.UnlockAndUpdate()
	m.urgentFunc = urgentFunc
	return m
}

var fs = afero.NewOsFs()

func (m *module) update() {
	bytes, err := afero.ReadFile(fs, m.thermalFile)
	if m.Error(err) {
		return
	}
	value := strings.TrimSpace(string(bytes))
	milliC, err := strconv.Atoi(value)
	if m.Error(err) {
		return
	}
	temp := Temperature(float64(milliC) / 1000.0)
	m.Lock()
	out := m.outputFunc(temp)
	if m.urgentFunc != nil {
		out.Urgent(m.urgentFunc(temp))
	}
	if m.colorFunc != nil {
		out.Color(m.colorFunc(temp))
	}
	m.Unlock()
	m.Output(out)
}
