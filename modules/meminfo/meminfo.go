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

	"github.com/dustin/go-humanize"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
	"github.com/soumya92/barista/modules/multi"
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

// Module represents a meminfo multi-module, and provides an interface
// for creating bar.Modules with various output functions/templates
// that share the same data source, cutting down on updates required.
type Module struct {
	moduleSet *multi.ModuleSet
	outputs   map[multi.Submodule]func(Info) bar.Output
}

// New constructs an instance of the meminfo multi-module
func New() *Module {
	m := &Module{
		moduleSet: multi.NewModuleSet(),
		outputs:   make(map[multi.Submodule]func(Info) bar.Output),
	}
	// Update meminfo when asked.
	m.moduleSet.OnUpdate(m.update)
	// Default is to refresh every 3s, matching the behaviour of top.
	m.RefreshInterval(3 * time.Second)
	return m
}

// RefreshInterval configures the polling frequency for meminfo.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.moduleSet.UpdateEvery(interval)
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
	i := make(Info)
	f, err := os.Open("/proc/meminfo")
	if m.moduleSet.Error(err) {
		return
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
		if m.moduleSet.Error(err) {
			return
		}
		i[name] = Bytes(intval << shift)
	}
	for submodule, outputFunc := range m.outputs {
		submodule.Output(outputFunc(i))
	}
}
