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
	"os"
	"time"

	"golang.org/x/sys/unix"

	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
)

// Info wraps disk space information.
type Info struct {
	Available unit.Datasize
	Free      unit.Datasize
	Total     unit.Datasize
}

// Used returns the disk space currently in use.
func (i Info) Used() unit.Datasize {
	return i.Total - i.Free
}

// UsedFrac returns the fraction of disk space currently in use.
func (i Info) UsedFrac() float64 {
	return float64(i.Used()) / float64(i.Total)
}

// UsedPct returns the percentage of disk space currently in use.
func (i Info) UsedPct() int {
	return int(i.UsedFrac()*100 + 0.5)
}

// AvailFrac returns the fraction of disk space available for use.
func (i Info) AvailFrac() float64 {
	return float64(i.Available) / float64(i.Total)
}

// AvailPct returns the percentage of disk space available for use.
func (i Info) AvailPct() int {
	return int(i.AvailFrac()*100 + 0.5)
}

// Module represents a diskspace bar module. It supports setting the output
// format, click handler, update frequency, and urgency/colour functions.
type Module struct {
	base.SimpleClickHandler
	path      string
	scheduler bar.Scheduler
	format    base.Value
}

type format struct {
	outputFunc func(Info) bar.Output
	colorFunc  func(Info) bar.Color
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

// New constructs an instance of the diskusage module for the given disk path.
func New(path string) *Module {
	m := &Module{
		path:      path,
		scheduler: base.Schedule().Every(3 * time.Second),
	}
	m.format.Set(format{})
	// Construct a simple template that's just 2 decimals of the used disk space.
	m.OutputTemplate(outputs.TextTemplate(`{{.Used.Gigabytes | printf "%.2f"}} GB`))
	return m
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *Module) OutputFunc(outputFunc func(Info) bar.Output) *Module {
	c := m.getFormat()
	c.outputFunc = outputFunc
	m.format.Set(c)
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *Module) OutputTemplate(template func(interface{}) bar.Output) *Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

// RefreshInterval configures the polling frequency for statfs.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// OutputColor configures a module to change the colour of its output based on a
// user-defined function. This allows you to set up color thresholds, or even
// blend between two colours based on the current disk utilisation.
func (m *Module) OutputColor(colorFunc func(Info) bar.Color) *Module {
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
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *Module) worker(ch base.Channel) {
	info, err := getStatFsInfo(m.path)
	format := m.getFormat()
	sFormat := m.format.Subscribe()
	for {
		if os.IsNotExist(err) {
			// Disk is not mounted, hide the module.
			// But continue regular updates so that the
			// disk is picked up on remount.
			ch.Output(outputs.Empty())
		} else {
			if ch.Error(err) {
				return
			}
			ch.Output(format.output(info))
		}
		select {
		case <-m.scheduler.Tick():
			info, err = getStatFsInfo(m.path)
		case <-sFormat.Tick():
			format = m.getFormat()
		}
	}
}

func getStatFsInfo(path string) (info Info, err error) {
	var statfsT unix.Statfs_t
	err = statfs(path, &statfsT)
	mult := unit.Datasize(statfsT.Bsize) * unit.Byte
	info.Available = unit.Datasize(statfsT.Bavail) * mult
	info.Free = unit.Datasize(statfsT.Bfree) * mult
	info.Total = unit.Datasize(statfsT.Blocks) * mult
	return // info, err
}

// To allow tests to mock out statfs.
var statfs = unix.Statfs
