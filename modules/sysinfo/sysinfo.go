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
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
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

// currentInfo stores the last value read by the updater.
// This allows newly created modules to start with data.
var currentInfo base.ErrorValue // of Info

var once sync.Once
var updater bar.Scheduler

// construct initialises sysinfo's global updating.
func construct() {
	once.Do(func() {
		updater = base.Schedule().Every(3 * time.Second)
		go func(updater bar.Scheduler) {
			for {
				update()
				updater.Wait()
			}
		}(updater)
	})
}

// RefreshInterval configures the polling frequency.
func RefreshInterval(interval time.Duration) {
	construct()
	updater.Every(interval)
}

// Module represents a bar.Module that displays memory information.
type Module struct {
	base.SimpleClickHandler
	ticker     bar.Ticker
	outputFunc base.Value
}

func defaultOutputFunc(i Info) bar.Output {
	return outputs.Textf("up: %s, load: %0.2f", i.Uptime, i.Loads[0])
}

// New creates a new sysinfo module.
func New() *Module {
	construct()
	m := &Module{ticker: currentInfo.Subscribe()}
	m.OutputFunc(defaultOutputFunc)
	return m
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *Module) OutputFunc(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *Module) OutputTemplate(template func(interface{}) bar.Output) *Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

// Stream subscribes to sysinfo and updates the module's output.
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *Module) worker(ch base.Channel) {
	i, err := currentInfo.Get()
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	sOutputFunc := m.outputFunc.Subscribe()
	for {
		if err != nil {
			ch.Output(outputs.Error(err))
		} else if info, ok := i.(Info); ok {
			ch.Output(outputFunc(info))
		}
		select {
		case <-sOutputFunc.Tick():
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
		case <-m.ticker.Tick():
			i, err = currentInfo.Get()
		}
	}
}

const loadScale = 65536.0 // LINUX_SYSINFO_LOADS_SCALE

func update() {
	var sysinfoT unix.Sysinfo_t
	err := unix.Sysinfo(&sysinfoT)
	if currentInfo.Error(err) {
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
	currentInfo.Set(sysinfo)
}
