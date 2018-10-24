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
package cputemp // import "barista.run/modules/cputemp"

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"

	"github.com/martinlindhe/unit"
	"github.com/spf13/afero"
)

// Module represents a cputemp bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module struct {
	thermalFile string
	scheduler   *timing.Scheduler
	outputFunc  value.Value // of func(unit.Temperature) bar.Output
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
	m.RefreshInterval(3 * time.Second)
	// Default output, if no function is specified later.
	m.Output(func(t unit.Temperature) bar.Output {
		return outputs.Textf("%.1f℃", t.Celsius())
	})
	return m
}

// OfType constructs an instance of the cputemp module for the *first* available
// sensor of the given type. "x86_pkg_temp" usually represents the temperature
// of the actual CPU package, while others may be available depending on the
// system, e.g. "iwlwifi" for wifi, or "acpitz" for the motherboard.
func OfType(typ string) *Module {
	files, _ := afero.ReadDir(fs, "/sys/class/thermal")
	for _, file := range files {
		name := file.Name()
		typFile := fmt.Sprintf("/sys/class/thermal/%s/type", name)
		typBytes, _ := afero.ReadFile(fs, typFile)
		if strings.TrimSpace(string(typBytes)) == typ {
			return Zone(name)
		}
	}
	return Zone("")
}

// New constructs an instance of the cputemp module for zone type "x86_pkg_temp".
// Returns nil of the x86_pkg_temp thermal zone is unavailable.
func New() *Module {
	return OfType("x86_pkg_temp")
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(unit.Temperature) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval configures the polling frequency for cpu temperatures.
// Note: updates might still be less frequent if the temperature does not change.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

var fs = afero.NewOsFs()

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	temp, err := getTemperature(m.thermalFile)
	outputFunc := m.outputFunc.Get().(func(unit.Temperature) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()
	for {
		if s.Error(err) {
			return
		}
		s.Output(outputFunc(temp))
		select {
		case <-m.scheduler.C:
			temp, err = getTemperature(m.thermalFile)
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(unit.Temperature) bar.Output)
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
