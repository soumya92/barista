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
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/modules/base"
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

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(LoadAvg) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(l LoadAvg) *bar.Output {
		// TODO: See if there's a way to avoid this.
		// Go does not agree with me when I say that a func(interface{})
		// should be assignable to a func(LoadAvg).
		return template(l)
	})
}

// RefreshInterval configures the polling frequency for getloadavg.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current load average.
type OutputColor func(LoadAvg) bar.Color

func (o OutputColor) apply(m *module) {
	m.colorFunc = o
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
type UrgentWhen func(LoadAvg) bool

func (u UrgentWhen) apply(m *module) {
	m.urgentFunc = u
}

// module is the type of the i3bar module. It is unexported because it's an
// implementation detail. It should never be used directly, only as something
// that satisfies the bar.Module interface.
type module struct {
	*base.Base
	refreshInterval time.Duration
	outputFunc      func(LoadAvg) *bar.Output
	colorFunc       func(LoadAvg) bar.Color
	urgentFunc      func(LoadAvg) bool
}

// New constructs an instance of the cpuload module with the provided configuration.
func New(config ...Config) base.Module {
	m := &module{
		Base: base.New(),
		// Default is to refresh every 3s, matching the behaviour of top.
		refreshInterval: 3 * time.Second,
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just 2 decimals of the 1-minute load average.
		defTpl := outputs.TextTemplate(`{{.Min1 | printf "%.2f"}}`)
		OutputTemplate(defTpl).apply(m)
	}
	// Worker goroutine to update load average at a fixed interval.
	m.SetWorker(m.loop)
	return m
}

func (m *module) loop() error {
	var loads LoadAvg
	for {
		count, err := C.getloadavg((*C.double)(&loads[0]), 3)
		if count != 3 {
			return fmt.Errorf("getloadavg: %d", count)
		}
		if err != nil {
			return err
		}
		out := m.outputFunc(loads)
		if m.urgentFunc != nil {
			out.Urgent = m.urgentFunc(loads)
		}
		if m.colorFunc != nil {
			out.Color = m.colorFunc(loads)
		}
		m.Output(out)
		time.Sleep(m.refreshInterval)
	}
}
