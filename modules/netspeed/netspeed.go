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

// Package netspeed provides an i3bar module to display network utilisation.
package netspeed

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
	"github.com/soumya92/barista/outputs"
)

// Speed represents unidirectional network traffic per second.
type Speed uint64

// In gets the speed in a specific unit, e.g. "b" or "MB".
func (s Speed) In(unit string) float64 {
	base, err := humanize.ParseBytes("1" + unit)
	if err != nil {
		base = 1
	}
	return float64(s) / float64(base)
}

// IEC returns the speed formatted in base 2.
func (s Speed) IEC() string {
	return humanize.IBytes(uint64(s))
}

// SI returns the speed formatted in base 10.
func (s Speed) SI() string {
	return humanize.Bytes(uint64(s))
}

// Speeds represents bidirectional network traffic.
type Speeds struct {
	Rx, Tx Speed
}

// Total gets the total speed (both up and down).
func (s Speeds) Total() Speed {
	return Speed(uint64(s.Rx) + uint64(s.Tx))
}

// Module represents a netspeed bar module. It supports setting the output
// format, click handler, and update frequency.
type Module interface {
	base.WithClickHandler
	RefreshInterval(time.Duration) Module
	OutputFunc(func(Speeds) bar.Output) Module
	OutputTemplate(func(interface{}) bar.Output) Module
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *module) OutputFunc(outputFunc func(Speeds) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(s Speeds) bar.Output {
		return template(s)
	})
}

// RefreshInterval configures the polling frequency for network speed.
// Since there is no concept of an instantaneous network speed, the speeds will
// be averaged over this interval before being displayed.
func (m *module) RefreshInterval(interval time.Duration) Module {
	m.scheduler.Stop()
	m.scheduler = m.UpdateEvery(interval)
	return m
}

// info represents that last read network information,
// and is used to compute the delta-rx and tx.
type info struct {
	Rx, Tx uint64
	Time   time.Time
}

// Refresh updates the last read information, and returns
// the delta-rx and tx since the last update in bytes/sec.
func (i *info) Refresh(rx, tx uint64) (dRx, dTx uint64) {
	duration := time.Since(i.Time).Seconds()
	dRx = uint64(float64(rx-i.Rx) / duration)
	dTx = uint64(float64(tx-i.Tx) / duration)
	i.Rx = rx
	i.Tx = tx
	i.Time = time.Now()
	return // dRx, dTx
}

type module struct {
	*base.Base
	rxFile, txFile string
	scheduler      base.Scheduler
	outputFunc     func(Speeds) bar.Output
	// To get network speed, we need to know delta-rx/tx,
	// so we need to store the previous rx/tx.
	lastRead info
}

// New constructs an instance of the netspeed module for the given interface.
func New(iface string) Module {
	m := &module{
		Base:   base.New(),
		rxFile: fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", iface),
		txFile: fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", iface),
	}
	// Default is to refresh every 3s, similar to top.
	m.RefreshInterval(3 * time.Second)
	// Default output template that's just the up and down speeds in SI.
	m.OutputTemplate(outputs.TextTemplate("{{.Tx.SI}}/s up | {{.Rx.SI}}/s down"))
	// Worker goroutine to recalculate speeds.
	m.OnUpdate(m.update)
	return m
}

func (m *module) update() {
	rx, erx := readFileAsUInt(m.rxFile)
	tx, etx := readFileAsUInt(m.txFile)
	if m.Error(erx) || m.Error(etx) {
		return
	}
	shouldOutput := !m.lastRead.Time.IsZero()
	dRx, dTx := m.lastRead.Refresh(rx, tx)
	if shouldOutput {
		m.Output(m.outputFunc(Speeds{
			Rx: Speed(dRx),
			Tx: Speed(dTx),
		}))
	}
}

func readFileAsUInt(file string) (uint64, error) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return 0, err
	}
	value := strings.TrimSpace(string(bytes))
	return strconv.ParseUint(value, 10, 64)
}
