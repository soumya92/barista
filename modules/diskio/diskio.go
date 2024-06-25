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

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/value"
	"github.com/soumya92/barista/format"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"

	"github.com/martinlindhe/unit"
	"github.com/spf13/afero"
)

// IO represents input and output rates for a disk.
type IO struct {
	Input, Output unit.Datarate
	// Unexported fields used by module to control output.
	shouldOutput bool
	err          error
}

// Total gets the total IO rate (input + output).
func (i IO) Total() unit.Datarate {
	return i.Input + i.Output
}

type diskInfo struct {
	ioChan     chan<- IO
	lastIO     *IO
	lastRead   uint64
	lastWrite  uint64
	updateTime time.Time
}

var once sync.Once

var lock sync.Mutex
var modules map[string]*diskInfo
var updater *timing.Scheduler

// construct initialises diskio's global updating. All diskio
// modules are updated with just one read of /proc/diskstats.
func construct() {
	once.Do(func() {
		modules = make(map[string]*diskInfo)
		updater = timing.NewScheduler()
		l.Attach(nil, updater, "diskio.updater")
		updater.Every(3 * time.Second)
		update()
		go func(updater *timing.Scheduler) {
			for range updater.C {
				update()
			}
		}(updater)
	})
}

// RefreshInterval configures the polling frequency.
func RefreshInterval(interval time.Duration) {
	construct()
	// Scheduler is goroutine safe, don't need to lock here.
	updater.Every(interval)
}

// Module represents a bar.Module for a single disk's io activity.
type Module struct {
	ioChan     <-chan IO
	outputFunc value.Value
}

// New creates a diskio module that displays disk io rates for the given disk.
func New(disk string) *Module {
	construct()
	lock.Lock()
	defer lock.Unlock()
	mInfo, found := modules[disk]
	if !found {
		mInfo = &diskInfo{}
		modules[disk] = mInfo
	}
	m := &Module{ioChan: mInfo.makeChannel()}
	l.Label(m, disk)
	l.Register(m, "ioChan", "outputFunc")
	m.Output(func(i IO) bar.Output {
		return outputs.Textf("Disk: %s", format.IByterate(i.Total()))
	})
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(IO) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Stream starts the module. Note that diskio updates begin as soon as the
// first module is constructed, even if no modules are streaming.
func (m *Module) Stream(s bar.Sink) {
	var i IO
	outputFunc := m.outputFunc.Get().(func(IO) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()
	for {
		select {
		case i = <-m.ioChan:
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(IO) bar.Output)
		}
		if s.Error(i.err) {
			continue
		}
		if i.shouldOutput {
			s.Output(outputFunc(i))
		} else {
			s.Output(nil)
		}
	}
}

// update updates the last read information, and returns
// the delta read and written since the last update in bytes/sec.
func (m *diskInfo) update(read, write uint64) (readRate, writeRate int) {
	duration := timing.Now().Sub(m.updateTime).Seconds()
	if read != m.lastRead {
		readRate = int(float64(read-m.lastRead) / duration)
	}
	if write != m.lastWrite {
		writeRate = int(float64(write-m.lastWrite) / duration)
	}
	m.lastRead = read
	m.lastWrite = write
	m.updateTime = timing.Now()
	return // readRate, writeRate
}

func (m *diskInfo) Error(err error) bool {
	if err == nil {
		return false
	}
	m.send(IO{err: err})
	return true
}

func (m *diskInfo) send(i IO) {
	if m.ioChan == nil {
		m.lastIO = &i
	} else {
		m.ioChan <- i
	}
}

func (m *diskInfo) makeChannel() <-chan IO {
	ioChan := make(chan IO, 1)
	m.ioChan = ioChan
	if m.lastIO != nil {
		m.ioChan <- *m.lastIO
	}
	return ioChan
}

var fs = afero.NewOsFs()

// To prevent data races in tests.
var signalChan chan bool

func update() {
	lock.Lock()
	defer lock.Unlock()
	var err error
	f, err := fs.Open("/proc/diskstats")
	if err != nil {
		for _, m := range modules {
			m.Error(err)
		}
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
		module, found := modules[disk]
		if !found {
			module = &diskInfo{}
			modules[disk] = module
		}
		updated[disk] = true
		reads, err := strconv.ParseUint(info[5], 10, 64)
		if module.Error(err) {
			continue
		}
		writes, err := strconv.ParseUint(info[9], 10, 64)
		if module.Error(err) {
			continue
		}
		shouldOutput := !module.updateTime.IsZero()
		readRate, writeRate := module.update(reads, writes)
		module.send(IO{
			// Linux always considers sectors to be 512 bytes long
			// independently of the devices real block size.
			// (from linux/types.h)
			Input:        unit.Datarate(readRate) * 512 * unit.BytePerSecond,
			Output:       unit.Datarate(writeRate) * 512 * unit.BytePerSecond,
			shouldOutput: shouldOutput,
		})
	}
	for disk, module := range modules {
		if !updated[disk] {
			module.lastRead = 0
			module.lastWrite = 0
			module.updateTime = time.Time{}
			module.send(IO{})
		}
	}
}
