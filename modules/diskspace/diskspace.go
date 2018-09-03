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
package diskspace // import "barista.run/modules/diskspace"

import (
	"os"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"

	"github.com/martinlindhe/unit"
	"golang.org/x/sys/unix"
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
	path       string
	scheduler  timing.Scheduler
	outputFunc value.Value // of func(Info) bar.Output
}

// New constructs an instance of the diskusage module for the given disk path.
func New(path string) *Module {
	m := &Module{
		path:      path,
		scheduler: timing.NewScheduler(),
	}
	l.Label(m, path)
	l.Register(m, "scheduler", "format")
	m.RefreshInterval(3 * time.Second)
	// Construct a simple output that's just 2 decimals of the used disk space.
	m.Output(func(i Info) bar.Output {
		return outputs.Textf("%.2f GB", i.Used().Gigabytes())
	})
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval configures the polling frequency for statfs.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	info, err := getStatFsInfo(m.path)
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	for {
		if os.IsNotExist(err) {
			// Disk is not mounted, hide the module.
			// But continue regular updates so that the
			// disk is picked up on remount.
			s.Output(nil)
		} else {
			if s.Error(err) {
				return
			}
			s.Output(outputFunc(info))
		}
		select {
		case <-m.scheduler.Tick():
			info, err = getStatFsInfo(m.path)
		case <-m.outputFunc.Next():
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
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
