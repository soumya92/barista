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

// Package diskio implements an i3bar module to show disk IO rates.
package diskio

import (
	"bufio"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/afero"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/multi"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
)

// Rate represents disk io in bytes per second.
type Rate uint64

// In gets the rate in a specific unit, e.g. "b" or "MB".
func (r Rate) In(unit string) float64 {
	base, err := humanize.ParseBytes("1" + unit)
	if err != nil {
		base = 1
	}
	return float64(r) / float64(base)
}

// IEC returns the rate formatted in base 2.
func (r Rate) IEC() string {
	return humanize.IBytes(uint64(r))
}

// SI returns the rate formatted in base 10.
func (r Rate) SI() string {
	return humanize.Bytes(uint64(r))
}

// IO represents input and output rates for a disk.
type IO struct {
	Input, Output Rate
}

// Total gets the total IO rate (input + output).
func (i IO) Total() Rate {
	return Rate(uint64(i.Input) + uint64(i.Output))
}

// Module represents a diskio multi-module, and provides an interface
// for creating bar.Modules for each disk that can display different output
// but fetch data from /proc/diskstats in one go.
type Module struct {
	sync.Mutex
	moduleSet  *multi.ModuleSet
	submodules map[string]*submodule
	scheduler  scheduler.Scheduler
	// Needed to prevent data race in tests.
	// Signals after the module is done reading /proc/diskstats.
	signalChan chan bool
}

// New constructs an instance of the diskio multi-module
func New() *Module {
	m := &Module{
		moduleSet:  multi.NewModuleSet(),
		submodules: make(map[string]*submodule),
	}
	// Update disk io rates when asked.
	m.moduleSet.OnUpdate(m.update)
	// Default is to refresh every 3s, matching the behaviour of top.
	m.scheduler = scheduler.Do(m.moduleSet.Update).Every(3 * time.Second)
	return m
}

// RefreshInterval configures the polling frequency.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	// Scheduler is goroutine safe, don't need to lock here.
	m.scheduler.Every(interval)
	return m
}

// Disk creates a submodule that displays disk io rates for the given disk.
func (m *Module) Disk(disk string) Submodule {
	m.Lock()
	defer m.Unlock()
	s := &submodule{
		Submodule: m.moduleSet.New(),
		parent:    m.moduleSet,
	}
	s.OutputTemplate(outputs.TextTemplate(`Disk: {{.Total.IEC}}/s`))
	m.submodules[disk] = s
	return s
}

// Submodule represents a bar.Module for a single disk's io activity.
type Submodule interface {
	base.WithClickHandler

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(IO) bar.Output) Submodule

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Submodule
}

// submodule wraps multi.Submodules and provides output formatting controls.
type submodule struct {
	multi.Submodule
	parent     *multi.ModuleSet
	outputFunc func(IO) bar.Output
	io         io
}

func (s *submodule) OutputFunc(outputFunc func(IO) bar.Output) Submodule {
	s.outputFunc = outputFunc
	s.Update()
	return s
}

func (s *submodule) OutputTemplate(template func(interface{}) bar.Output) Submodule {
	return s.OutputFunc(func(i IO) bar.Output {
		return template(i)
	})
}

// io represents that last read disk io counters,
// and is used to compute the rate of disk io.
type io struct {
	In, Out uint64
	Time    time.Time
}

// Update updates the last read information, and returns
// the delta read and written since the last update in bytes/sec.
func (i *io) Update(in, out uint64) (inRate, outRate int) {
	duration := scheduler.Now().Sub(i.Time).Seconds()
	inRate = int(float64(in-i.In) / duration)
	outRate = int(float64(out-i.Out) / duration)
	i.In = in
	i.Out = out
	i.Time = scheduler.Now()
	return // inRate, outRate
}

var fs = afero.NewOsFs()

func (m *Module) update() {
	m.Lock()
	defer m.Unlock()
	var err error
	f, err := fs.Open("/proc/diskstats")
	if m.moduleSet.Error(err) {
		return
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	// Keep track of which submodules were updated, so that any drives
	// that were removed can be cleared instead of showing stale data.
	updated := make(map[string]bool)
	for s.Scan() {
		info := strings.Fields(s.Text())
		if len(info) < 14 {
			continue
		}
		// See https://www.kernel.org/doc/Documentation/iostats.txt
		disk := info[2]
		submodule, found := m.submodules[disk]
		if !found {
			// Don't care about this disk
			continue
		}
		updated[disk] = true
		reads, err := strconv.ParseUint(info[5], 10, 64)
		if submodule.Error(err) {
			continue
		}
		writes, err := strconv.ParseUint(info[9], 10, 64)
		if submodule.Error(err) {
			continue
		}
		shouldOutput := !submodule.io.Time.IsZero()
		readRate, writeRate := submodule.io.Update(reads, writes)
		if shouldOutput {
			// Linux always considers sectors to be 512 bytes long
			// independently of the devices real block size.
			// (from linux/types.h)
			submodule.Output(submodule.outputFunc(IO{
				Input:  Rate(readRate * 512),
				Output: Rate(writeRate * 512),
			}))
		}
	}
	for disk, submodule := range m.submodules {
		if !updated[disk] {
			if !submodule.io.Time.IsZero() {
				submodule.Clear()
				submodule.io = io{}
			}
		}
	}
	if m.signalChan != nil {
		m.signalChan <- true
	}
}
