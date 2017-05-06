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

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(Speeds) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(s Speeds) *bar.Output {
		return template(s)
	})
}

// Interface sets the name of the interface to display speeds for.
type Interface string

func (i Interface) apply(m *module) {
	m.rxFile = fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", string(i))
	m.txFile = fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", string(i))
}

// RefreshInterval configures the polling frequency for network speed.
// Since there is no concept of an instantaneous network speed, the speeds will
// be averaged over this interval before being displayed.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
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
	rxFile, txFile  string
	refreshInterval time.Duration
	outputFunc      func(Speeds) *bar.Output
	// To get network speed, we need to know delta-rx/tx,
	// so we need to store the previous rx/tx.
	lastRead info
}

// New constructs an instance of the netspeed module with the provided configuration.
func New(config ...Config) base.WithClickHandler {
	m := &module{
		Base: base.New(),
		// Default is to refresh every 3s, similar to top.
		refreshInterval: 3 * time.Second,
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just the up and down speeds in SI.
		defTpl := outputs.TextTemplate("{{.Tx.SI}}/s up | {{.Rx.SI}}/s down")
		OutputTemplate(defTpl).apply(m)
	}
	// Default interface for goobuntu workstations, if not specified.
	if m.rxFile == "" {
		Interface("em1").apply(m)
	}
	// Worker goroutine to recalculate speeds.
	m.OnUpdate(m.update)
	m.UpdateEvery(m.refreshInterval)
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
