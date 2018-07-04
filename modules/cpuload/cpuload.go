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
	"image/color"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
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
type Module struct {
	base.SimpleClickHandler
	scheduler timing.Scheduler
	format    base.Value
}

type format struct {
	outputFunc func(LoadAvg) bar.Output
	colorFunc  func(LoadAvg) color.Color
	urgentFunc func(LoadAvg) bool
}

func (f format) output(l LoadAvg) bar.Output {
	out := outputs.Group(f.outputFunc(l))
	if f.urgentFunc != nil {
		out.Urgent(f.urgentFunc(l))
	}
	if f.colorFunc != nil {
		out.Color(f.colorFunc(l))
	}
	return out
}

func (m *Module) getFormat() format {
	return m.format.Get().(format)
}

// New constructs an instance of the cpuload module.
func New() *Module {
	m := &Module{scheduler: timing.NewScheduler()}
	l.Register(m, "scheduler", "format")
	m.format.Set(format{})
	m.RefreshInterval(3 * time.Second)
	// Construct a simple template that's just 2 decimals of the 1-minute load average.
	m.Template(`{{.Min1 | printf "%.2f"}}`)
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(LoadAvg) bar.Output) *Module {
	c := m.getFormat()
	c.outputFunc = outputFunc
	m.format.Set(c)
	return m
}

// Template configures a module to display the output of a template.
func (m *Module) Template(template string) *Module {
	templateFn := outputs.TextTemplate(template)
	return m.Output(func(l LoadAvg) bar.Output {
		// TODO: See if there's a way to avoid this.
		// Go does not agree with me when I say that a func(interface{})
		// should be assignable to a func(LoadAvg).
		return templateFn(l)
	})
}

// RefreshInterval configures the polling frequency for getloadavg.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current load average.
func (m *Module) OutputColor(colorFunc func(LoadAvg) color.Color) *Module {
	c := m.getFormat()
	c.colorFunc = colorFunc
	m.format.Set(c)
	return m
}

// UrgentWhen configures a module to mark its output as urgent based on a
// user-defined function.
func (m *Module) UrgentWhen(urgentFunc func(LoadAvg) bool) *Module {
	c := m.getFormat()
	c.urgentFunc = urgentFunc
	m.format.Set(c)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	var loads LoadAvg
	count, err := getloadavg(&loads, 3)
	format := m.getFormat()
	for {
		if s.Error(err) {
			return
		}
		if count != 3 {
			s.Error(fmt.Errorf("getloadavg: %d", count))
			return
		}
		s.Output(format.output(loads))
		select {
		case <-m.scheduler.Tick():
			count, err = getloadavg(&loads, 3)
		case <-m.format.Update():
			format = m.getFormat()
		}
	}
}

// To allow tests to mock out getloadavg.
var getloadavg = func(out *LoadAvg, count int) (int, error) {
	read, err := C.getloadavg((*C.double)(&out[0]), (C.int)(count))
	return int(read), err
}
