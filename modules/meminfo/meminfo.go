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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/martinlindhe/unit"
	"github.com/spf13/afero"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
)

// Info wraps meminfo output.
// See /proc/meminfo for names of keys.
// Some common functions are also provided.
type Info map[string]unit.Datasize

// FreeFrac returns a free/total metric for a given name,
// e.g. Mem, Swap, High, etc.
func (i Info) FreeFrac(k string) float64 {
	return float64(i[k+"Free"]) / float64(i[k+"Total"])
}

// Available returns the "available" system memory, including
// currently cached memory that can be freed up if needed.
func (i Info) Available() unit.Datasize {
	// MemAvailable, if present, is a more accurate indication of
	// available memory.
	if avail, ok := i["MemAvailable"]; ok {
		return avail
	}
	return i["MemFree"] + i["Cached"] + i["Buffers"]
}

// AvailFrac returns the available memory as a fraction of total.
func (i Info) AvailFrac() float64 {
	return float64(i.Available()) / float64(i["MemTotal"])
}

// currentInfo stores the last value read by the updater.
// This allows newly created modules to start with data.
var currentInfo base.ErrorValue // of Info
// infoEmitter informs all registered modules about updates
// to currentInfo.
var infoEmitter *base.Emitter

var once sync.Once
var updater timing.Scheduler

// construct initialises meminfo's global updating. All meminfo
// modules are updated with just one read of /proc/meminfo.
func construct() {
	once.Do(func() {
		updater = timing.NewScheduler()
		infoEmitter = base.Multicast(currentInfo.Update())
		l.Attach(nil, &currentInfo, "meminfo.currentInfo")
		l.Attach(nil, updater, "meminfo.updater")
		l.Attach(&currentInfo, &infoEmitter, ".emitter")
		updater.Every(3 * time.Second)
		go func(updater timing.Scheduler) {
			for {
				update()
				<-updater.Tick()
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
	ticker     <-chan struct{}
	outputFunc base.Value
}

func defaultOutputFunc(i Info) bar.Output {
	return outputs.Textf("Mem: %s", outputs.IBytesize(i.Available()))
}

// New creates a new meminfo module.
func New() *Module {
	construct()
	m := &Module{ticker: infoEmitter.Subscribe()}
	l.Register(m, "outputFunc")
	m.OutputFunc(defaultOutputFunc)
	return m
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *Module) OutputFunc(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *Module) OutputTemplate(template string) *Module {
	templateFn := outputs.TextTemplate(template)
	return m.OutputFunc(func(i Info) bar.Output {
		return templateFn(i)
	})
}

// Stream subscribes to meminfo and updates the module's output accordingly.
func (m *Module) Stream(s bar.Sink) {
	i, err := currentInfo.Get()
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	for {
		if err != nil {
			s.Error(err)
		} else if info, ok := i.(Info); ok {
			s.Output(outputFunc(info))
		}
		select {
		case <-m.outputFunc.Update():
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
		case <-m.ticker:
			i, err = currentInfo.Get()
		}
	}
}

var fs = afero.NewOsFs()

func update() {
	info := make(Info)
	f, err := fs.Open("/proc/meminfo")
	if currentInfo.Error(err) {
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
		mult := unit.Byte
		// 0 values may not have kB, but kB is the only possible unit here.
		// see sysinfo.c from psprocs, where everything is also assumed to be kb.
		if strings.HasSuffix(value, " kB") {
			mult = unit.Kibibyte
			value = value[:len(value)-len(" kB")]
		}
		if intval, err := strconv.ParseUint(value, 10, 64); err == nil {
			info[name] = unit.Datasize(intval) * mult
		}
	}
	currentInfo.Set(info)
}
