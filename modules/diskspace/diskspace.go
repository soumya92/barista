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

// Package diskspace provides an i3bar module for disk space usage.
package diskspace

import (
	"syscall"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
)

// Info wraps disk space information.
type Info struct {
	Available Bytes
	Free      Bytes
	Total     Bytes
}

// Used returns the disk space currently in use.
func (i Info) Used() Bytes {
	return Bytes(int(i.Total) - int(i.Free))
}

// UsedFrac returns the fraction of disk space currently in use.
func (i Info) UsedFrac() float64 {
	return float64(i.Used()) / float64(i.Total)
}

// UsedPct returns the percentage of disk space currently in use.
func (i Info) UsedPct() int {
	return int(i.UsedFrac() * 100)
}

// AvailFrac returns the fraction of disk space available for use.
func (i Info) AvailFrac() float64 {
	return float64(i.Available) / float64(i.Total)
}

// AvailPct returns the percentage of disk space available for use.
func (i Info) AvailPct() int {
	return int(i.AvailFrac() * 100)
}

// Bytes represents a size in bytes.
type Bytes uint64

// In gets the size in a specific unit, e.g. "b" or "MB".
func (b Bytes) In(unit string) float64 {
	base, err := humanize.ParseBytes("1" + unit)
	if err != nil {
		base = 1
	}
	return float64(b) / float64(base)
}

// IEC returns the size formatted in base 2.
func (b Bytes) IEC() string {
	return humanize.IBytes(uint64(b))
}

// SI returns the size formatted in base 10.
func (b Bytes) SI() string {
	return humanize.Bytes(uint64(b))
}

// Module represents a diskspace bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module interface {
	base.WithClickHandler

	// RefreshInterval configures the polling frequency for statfs.
	RefreshInterval(time.Duration) Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Info) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OutputColor configures a module to change the colour of its output based on a
	// user-defined function. This allows you to set up color thresholds, or even
	// blend between two colours based on the current disk utilisation.
	OutputColor(func(Info) bar.Color) Module

	// UrgentWhen configures a module to mark its output as urgent based on a
	// user-defined function.
	UrgentWhen(func(Info) bool) Module
}

type module struct {
	*base.Base
	path       string
	scheduler  scheduler.Scheduler
	outputFunc func(Info) bar.Output
	colorFunc  func(Info) bar.Color
	urgentFunc func(Info) bool
	statResult syscall.Statfs_t
}

// New constructs an instance of the diskusage module for the given disk path.
func New(path string) Module {
	m := &module{
		Base: base.New(),
		path: path,
	}
	// Default is to refresh every 3s, matching the behaviour of top.
	m.scheduler = scheduler.Do(m.Update).Every(3 * time.Second)
	// Construct a simple template that's just 2 decimals of the used disk space.
	m.OutputTemplate(outputs.TextTemplate(`{{.Used.In "GB" | printf "%.2f"}} GB`))
	// Update disk information when asked.
	m.OnUpdate(m.update)
	return m
}

func (m *module) OutputFunc(outputFunc func(Info) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

func (m *module) RefreshInterval(interval time.Duration) Module {
	m.scheduler.Every(interval)
	return m
}

func (m *module) OutputColor(colorFunc func(Info) bar.Color) Module {
	m.colorFunc = colorFunc
	m.Update()
	return m
}

func (m *module) UrgentWhen(urgentFunc func(Info) bool) Module {
	m.urgentFunc = urgentFunc
	m.Update()
	return m
}

func (m *module) update() {
	if m.Error(syscall.Statfs(m.path, &m.statResult)) {
		return
	}
	mult := uint64(m.statResult.Bsize)
	info := Info{
		Available: Bytes(m.statResult.Bavail * mult),
		Free:      Bytes(m.statResult.Bfree * mult),
		Total:     Bytes(m.statResult.Blocks * mult),
	}
	out := m.outputFunc(info)
	if m.urgentFunc != nil {
		out.Urgent(m.urgentFunc(info))
	}
	if m.colorFunc != nil {
		out.Color(m.colorFunc(info))
	}
	m.Output(out)
}
