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

// Package sysinfo implements i3bar modules that show system information.
package sysinfo

import (
	"time"

	"golang.org/x/sys/unix"

	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/multi"
	"github.com/soumya92/barista/base/scheduler"
)

// Info wraps the result of sysinfo and makes it more useful.
type Info struct {
	Uptime       time.Duration
	Loads        [3]float64
	TotalRAM     unit.Datasize
	FreeRAM      unit.Datasize
	SharedRAM    unit.Datasize
	BufferRAM    unit.Datasize
	TotalSwap    unit.Datasize
	FreeSwap     unit.Datasize
	Procs        uint16
	TotalHighRAM unit.Datasize
	FreeHighRAM  unit.Datasize
}

// Module represents a sysinfo multi-module, and provides an interface
// for creating bar.Modules with various output functions/templates
// that share the same data source, cutting down on updates required.
type Module struct {
	moduleSet *multi.ModuleSet
	outputs   map[multi.Submodule]func(Info) bar.Output
	scheduler scheduler.Scheduler
}

// New constructs an instance of the sysinfo multi-module
func New() *Module {
	m := &Module{
		moduleSet: multi.NewModuleSet(),
		// Because the nil value of map is not sensible :(
		outputs: make(map[multi.Submodule]func(Info) bar.Output),
	}
	// Update sysinfo when asked.
	m.moduleSet.OnUpdate(m.update)
	// Default is to refresh every 3s, matching the behaviour of top.
	m.scheduler = scheduler.Do(m.moduleSet.Update).Every(3 * time.Second)
	return m
}

// RefreshInterval configures the polling frequency.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// OutputFunc creates a submodule that displays the output of a user-defined function.
func (m *Module) OutputFunc(format func(Info) bar.Output) base.WithClickHandler {
	submodule := m.moduleSet.New()
	m.outputs[submodule] = format
	return submodule
}

// OutputTemplate creates a submodule that displays the output of a template.
func (m *Module) OutputTemplate(template func(interface{}) bar.Output) base.WithClickHandler {
	return m.OutputFunc(func(i Info) bar.Output { return template(i) })
}

func (m *Module) update() {
	var sysinfoT unix.Sysinfo_t
	const loadScale = 65536.0 // LINUX_SYSINFO_LOADS_SCALE
	if m.moduleSet.Error(unix.Sysinfo(&sysinfoT)) {
		return
	}
	mult := unit.Datasize(sysinfoT.Unit) * unit.Byte
	sysinfo := Info{
		Uptime: time.Duration(sysinfoT.Uptime) * time.Second,
		Loads: [3]float64{
			float64(sysinfoT.Loads[0]) / loadScale,
			float64(sysinfoT.Loads[1]) / loadScale,
			float64(sysinfoT.Loads[2]) / loadScale,
		},
		Procs:        sysinfoT.Procs,
		TotalRAM:     unit.Datasize(sysinfoT.Totalram) * mult,
		FreeRAM:      unit.Datasize(sysinfoT.Freeram) * mult,
		SharedRAM:    unit.Datasize(sysinfoT.Sharedram) * mult,
		BufferRAM:    unit.Datasize(sysinfoT.Bufferram) * mult,
		TotalSwap:    unit.Datasize(sysinfoT.Totalswap) * mult,
		FreeSwap:     unit.Datasize(sysinfoT.Freeswap) * mult,
		TotalHighRAM: unit.Datasize(sysinfoT.Totalhigh) * mult,
		FreeHighRAM:  unit.Datasize(sysinfoT.Freehigh) * mult,
	}
	for submodule, outputFunc := range m.outputs {
		submodule.Output(outputFunc(sysinfo))
	}
}
