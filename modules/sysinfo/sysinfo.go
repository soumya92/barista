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
	"syscall"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/multi"

	"github.com/dustin/go-humanize"
)

// Info wraps the result of sysinfo and makes it more useful.
type Info struct {
	Uptime       time.Duration
	Loads        [3]float64
	TotalRAM     Bytes
	FreeRAM      Bytes
	SharedRAM    Bytes
	BufferRAM    Bytes
	TotalSwap    Bytes
	FreeSwap     Bytes
	Procs        uint16
	TotalHighRAM Bytes
	FreeHighRAM  Bytes
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

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

type outputFunc struct {
	Key    interface{}
	Format func(Info) *bar.Output
}

func (o outputFunc) apply(m *module) {
	m.ModuleSet.Add(o.Key)
	m.outputFuncs[o.Key] = o.Format
}

// OutputFunc configures a module to display the output of a user-defined function.
func OutputFunc(key interface{}, format func(Info) *bar.Output) Config {
	return outputFunc{
		Key:    key,
		Format: format,
	}
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(key interface{}, template func(interface{}) *bar.Output) Config {
	return outputFunc{
		Key: key,
		Format: func(i Info) *bar.Output {
			return template(i)
		},
	}
}

// RefreshInterval configures the polling frequency for sysinfo.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
}

// module is the type of the i3bar module. It is unexported because it's an
// implementation detail. It should never be used directly, only as something
// that satisfies the bar.Module interface.
type module struct {
	*multi.ModuleSet
	refreshInterval time.Duration
	outputFuncs     map[interface{}]func(Info) *bar.Output
}

// New constructs an instance of the sysinfo multi-module
// with the provided configuration.
func New(config ...Config) multi.Module {
	m := &module{
		ModuleSet: multi.NewModuleSet(),
		// Default is to refresh every 3s, matching the behaviour of top.
		refreshInterval: 3 * time.Second,
		// Because the nil value of map is not sensible :(
		outputFuncs: make(map[interface{}]func(Info) *bar.Output),
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Worker goroutine to update sysinfo at a fixed interval.
	m.SetWorker(m.loop)
	return m
}

func (m *module) loop() error {
	var sysinfoT syscall.Sysinfo_t
	const loadScale = 65536.0 // LINUX_SYSINFO_LOADS_SCALE
	for {
		if err := syscall.Sysinfo(&sysinfoT); err != nil {
			return err
		}
		unit := uint64(sysinfoT.Unit)
		sysinfo := Info{
			Uptime: time.Duration(sysinfoT.Uptime) * time.Second,
			Loads: [3]float64{
				float64(sysinfoT.Loads[0]) / loadScale,
				float64(sysinfoT.Loads[1]) / loadScale,
				float64(sysinfoT.Loads[2]) / loadScale,
			},
			Procs:        sysinfoT.Procs,
			TotalRAM:     Bytes(sysinfoT.Totalram * unit),
			FreeRAM:      Bytes(sysinfoT.Freeram * unit),
			SharedRAM:    Bytes(sysinfoT.Sharedram * unit),
			BufferRAM:    Bytes(sysinfoT.Bufferram * unit),
			TotalSwap:    Bytes(sysinfoT.Totalswap * unit),
			FreeSwap:     Bytes(sysinfoT.Freeswap * unit),
			TotalHighRAM: Bytes(sysinfoT.Totalhigh * unit),
			FreeHighRAM:  Bytes(sysinfoT.Freehigh * unit),
		}
		for key, outputFunc := range m.outputFuncs {
			m.Output(key, outputFunc(sysinfo))
		}
		time.Sleep(m.refreshInterval)
	}
}
