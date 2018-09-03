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

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"
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
	scheduler  timing.Scheduler
	outputFunc value.Value // of func(LoadAvg) bar.Output
}

// New constructs an instance of the cpuload module.
func New() *Module {
	m := &Module{scheduler: timing.NewScheduler()}
	l.Register(m, "scheduler", "format")
	m.RefreshInterval(3 * time.Second)
	// Construct a simple output that's just 2 decimals of the 1-minute load average.
	m.Output(func(l LoadAvg) bar.Output {
		return outputs.Textf("%.2f", l.Min1())
	})
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(LoadAvg) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval configures the polling frequency for getloadavg.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	var loads LoadAvg
	count, err := getloadavg(&loads, 3)
	outputFunc := m.outputFunc.Get().(func(LoadAvg) bar.Output)
	for {
		if s.Error(err) {
			return
		}
		if count != 3 {
			s.Error(fmt.Errorf("getloadavg: %d", count))
			return
		}
		s.Output(outputFunc(loads))
		select {
		case <-m.scheduler.Tick():
			count, err = getloadavg(&loads, 3)
		case <-m.outputFunc.Next():
			outputFunc = m.outputFunc.Get().(func(LoadAvg) bar.Output)
		}
	}
}

// To allow tests to mock out getloadavg.
var getloadavg = func(out *LoadAvg, count int) (int, error) {
	read, err := C.getloadavg((*C.double)(&out[0]), (C.int)(count))
	return int(read), err
}
