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

// Package cpuload implements an i3bar module that shows load averages.
// Deprecated in favour of SysInfo, which can show more than just load average.
package cpuload

//#include <stdlib.h>
import "C"
import (
	"fmt"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
)

// LoadAvg represents the CPU load average for the past 1, 5, and 15 minutes.
type LoadAvg [3]float64

// Min1 returns the CPU load average for the past 1 minute.
func (l LoadAvg) Min1() float64 {
	return l[0]
}

// Min5 returns the CPU load average for the past 5 minutes.
func (l LoadAvg) Min5() float64 {
	return l[1]
}

// Min15 returns the CPU load average for the past 15 minutes.
func (l LoadAvg) Min15() float64 {
	return l[2]
}

// Module represents a cpuload bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module interface {
	base.WithClickHandler

	// RefreshInterval configures the polling frequency for getloadavg.
	RefreshInterval(time.Duration) Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(LoadAvg) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OutputColor configures a module to change the colour of its output based on a
	// user-defined function. This allows you to set up color thresholds, or even
	// blend between two colours based on the current load average.
	OutputColor(func(LoadAvg) bar.Color) Module

	// UrgentWhen configures a module to mark its output as urgent based on a
	// user-defined function.
	UrgentWhen(func(LoadAvg) bool) Module
}

type module struct {
	*base.Base
	outputFunc func(LoadAvg) bar.Output
	colorFunc  func(LoadAvg) bar.Color
	urgentFunc func(LoadAvg) bool
	loads      LoadAvg
}

// New constructs an instance of the cpuload module.
func New() Module {
	m := &module{Base: base.New()}
	// Default is to refresh every 3s, matching the behaviour of top.
	m.Schedule().Every(3 * time.Second)
	// Construct a simple template that's just 2 decimals of the 1-minute load average.
	m.OutputTemplate(outputs.TextTemplate(`{{.Min1 | printf "%.2f"}}`))
	// Update load average when asked.
	m.OnUpdate(m.update)
	return m
}

func (m *module) OutputFunc(outputFunc func(LoadAvg) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(l LoadAvg) bar.Output {
		// TODO: See if there's a way to avoid this.
		// Go does not agree with me when I say that a func(interface{})
		// should be assignable to a func(LoadAvg).
		return template(l)
	})
}

func (m *module) RefreshInterval(interval time.Duration) Module {
	m.Schedule().Every(interval)
	return m
}

func (m *module) OutputColor(colorFunc func(LoadAvg) bar.Color) Module {
	m.colorFunc = colorFunc
	m.Update()
	return m
}

func (m *module) UrgentWhen(urgentFunc func(LoadAvg) bool) Module {
	m.urgentFunc = urgentFunc
	m.Update()
	return m
}

func (m *module) update() {
	count, err := C.getloadavg((*C.double)(&m.loads[0]), 3)
	if count != 3 {
		m.Error(fmt.Errorf("getloadavg: %d", count))
		return
	}
	if m.Error(err) {
		return
	}
	out := m.outputFunc(m.loads)
	if m.urgentFunc != nil {
		out.Urgent(m.urgentFunc(m.loads))
	}
	if m.colorFunc != nil {
		out.Color(m.colorFunc(m.loads))
	}
	m.Output(out)
}
