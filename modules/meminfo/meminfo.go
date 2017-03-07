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

// Package meminfo provides an i3bar module that shows memory information.
package meminfo

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/multi"

	"github.com/dustin/go-humanize"
)

// Info wraps meminfo output.
// See /proc/meminfo for names of keys.
// Some common functions are also provided.
type Info map[string]Bytes

// FreeFrac returns a free/total metric for a given name,
// e.g. Mem, Swap, High, etc.
func (i Info) FreeFrac(k string) float64 {
	return float64(i[k+"Free"]) / float64(i[k+"Total"])
}

// Available returns the "available" system memory, including
// currently cached memory that can be freed up if needed.
func (i Info) Available() Bytes {
	return Bytes(uint64(i["MemFree"]) + uint64(i["Cached"]) + uint64(i["Buffers"]))
}

// AvailFrac returns the available memory as a fraction of total.
func (i Info) AvailFrac() float64 {
	return float64(i.Available()) / float64(i["MemTotal"])
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
		outputFuncs:     make(map[interface{}]func(Info) *bar.Output),
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
	for {
		i, err := memInfo()
		if err != nil {
			return err
		}
		for key, outputFunc := range m.outputFuncs {
			m.Output(key, outputFunc(i))
		}
		time.Sleep(m.refreshInterval)
	}
}

func memInfo() (Info, error) {
	i := make(Info)
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}
		name := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		var shift uint
		// 0 values may not have kB, but kB is the only possible unit here.
		// see sysinfo.c from psprocs, where everything is also assumed to be kb.
		if strings.HasSuffix(value, " kB") {
			shift = 10
			value = value[:len(value)-len(" kB")]
		}
		intval, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return nil, err
		}
		i[name] = Bytes(intval << shift)
	}
	return i, nil
}
