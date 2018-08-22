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
	"image/color"
	"strconv"
	"strings"
	"time"

	"github.com/martinlindhe/unit"
	"github.com/spf13/afero"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
)

// Module represents a cputemp bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module struct {
	thermalFile string
	scheduler   timing.Scheduler
	format      base.Value
}

type format struct {
	outputFunc func(unit.Temperature) bar.Output
	colorFunc  func(unit.Temperature) color.Color
	urgentFunc func(unit.Temperature) bool
}

func (f format) output(t unit.Temperature) bar.Output {
	out := outputs.Group(f.outputFunc(t))
	if f.urgentFunc != nil {
		out.Urgent(f.urgentFunc(t))
	}
	if f.colorFunc != nil {
		out.Color(f.colorFunc(t))
	}
	return out
}

func (m *Module) getFormat() format {
	return m.format.Get().(format)
}

// Zone constructs an instance of the cputemp module for the specified zone.
// The file /sys/class/thermal/<zone>/temp should return cpu temp in 1/1000 deg C.
func Zone(thermalZone string) *Module {
	m := &Module{
		thermalFile: fmt.Sprintf("/sys/class/thermal/%s/temp", thermalZone),
		scheduler:   timing.NewScheduler(),
	}
	l.Label(m, thermalZone)
	l.Register(m, "scheduler", "format")
	m.format.Set(format{})
	m.RefreshInterval(3 * time.Second)
	// Default output template, if no template/function was specified.
	m.Template(`{{.Celsius | printf "%.1f"}}℃`)
	return m
}

// DefaultZone constructs an instance of the cputemp module for the default zone.
func DefaultZone() *Module {
	return Zone("thermal_zone0")
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(unit.Temperature) bar.Output) *Module {
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

// RefreshInterval configures the polling frequency for cpu temperatures.
// Note: updates might still be less frequent if the temperature does not change.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current temperature.
func (m *Module) OutputColor(colorFunc func(unit.Temperature) color.Color) *Module {
	c := m.getFormat()
	c.colorFunc = colorFunc
	m.format.Set(c)
	return m
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
func (m *Module) UrgentWhen(urgentFunc func(unit.Temperature) bool) *Module {
	c := m.getFormat()
	c.urgentFunc = urgentFunc
	m.format.Set(c)
	return m
}

var fs = afero.NewOsFs()

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	temp, err := getTemperature(m.thermalFile)
	format := m.getFormat()
	for {
		if s.Error(err) {
			return
		}
		s.Output(format.output(temp))
		select {
		case <-m.scheduler.Tick():
			temp, err = getTemperature(m.thermalFile)
		case <-m.format.Update():
			format = m.getFormat()
		}
	}
}

func getTemperature(thermalFile string) (unit.Temperature, error) {
	bytes, err := afero.ReadFile(fs, thermalFile)
	if err != nil {
		return 0, err
	}
	value := strings.TrimSpace(string(bytes))
	milliC, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return unit.FromCelsius(float64(milliC) / 1000.0), nil
}
