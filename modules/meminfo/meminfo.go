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

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
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

var once sync.Once
var updater bar.Scheduler

// construct initialises meminfo's global updating. All meminfo
// modules are updated with just one read of /proc/meminfo.
func construct() {
	once.Do(func() {
		updater = barista.Schedule().Every(3 * time.Second)
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
type Module interface {
	base.SimpleClickHandlerModule

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Info) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module
}

type module struct {
	base.SimpleClickHandler
	ticker     bar.Ticker
	outputFunc base.Value
}

func defaultOutputFunc(i Info) bar.Output {
	return outputs.Textf("Mem: %s", outputs.IBytesize(i.Available()))
}

// New creates a new meminfo module.
func New() Module {
	construct()
	m := &module{ticker: currentInfo.Subscribe()}
	m.OutputFunc(defaultOutputFunc)
	return m
}

func (m *module) OutputFunc(outputFunc func(Info) bar.Output) Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

func (m *module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *module) worker(ch base.Channel) {
	i, err := currentInfo.Get()
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	sOutputFunc := m.outputFunc.Subscribe()
	for {
		if err != nil {
			// Do not use ch.Error because that will close the channel,
			// leaving further updates to deadlock.
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
