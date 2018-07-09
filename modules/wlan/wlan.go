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

// Package wlan provides an i3bar module for wireless information.
// NOTE: This module REQUIRES the external command "iwgetid",
// because getting the SSID is a privileged operation.
package wlan

import (
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/watchers/netlink"
	l "github.com/soumya92/barista/logging"
)

// Info represents the wireless card status.
type Info struct {
	State          netlink.OperState
	IPs            []net.IP
	SSID           string
	AccessPointMAC string
	Channel        int
	Frequency      float64
}

// Connecting returns true if a connection is in progress.
func (i Info) Connecting() bool {
	return i.State == netlink.Dormant
}

// Connected returns true if connected to a wireless network.
func (i Info) Connected() bool {
	return i.State == netlink.Up
}

// Enabled returns true if the wireless card is enabled.
func (i Info) Enabled() bool {
	return i.State != netlink.Gone && i.State != netlink.NotPresent
}

// Module represents a wlan bar module.
type Module struct {
	base.SimpleClickHandler
	intf       string
	outputFunc base.Value // of func(Info) bar.Output
}

// New constructs an instance of the wlan module for the specified interface.
func New(iface string) *Module {
	m := &Module{intf: iface}
	l.Label(m, iface)
	l.Register(m, "outputFunc")
	// Default output template is just the SSID when connected.
	m.Template("{{if .Connected}}{{.SSID}}{{end}}")
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Template configures a module to display the output of a template.
func (m *Module) Template(template string) *Module {
	base.Template(template, m.Output)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	info := Info{State: netlink.Gone}
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	updateChan := netlink.ByName(m.intf)
	for {
		select {
		case update := <-updateChan:
			info.State = update.State
			info.IPs = update.IPs
			m.fillWifiInfo(&info)
		case <-m.outputFunc.Update():
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
		}
		s.Output(outputFunc(info))
	}
}

func (m *Module) fillWifiInfo(info *Info) {
	ssid, err := iwgetid(m.intf, "-r")
	if err != nil {
		return
	}
	info.SSID = ssid
	info.AccessPointMAC, _ = iwgetid(m.intf, "-a")
	ch, _ := iwgetid(m.intf, "-c")
	info.Channel, _ = strconv.Atoi(ch)
	freq, _ := iwgetid(m.intf, "-f")
	info.Frequency, _ = strconv.ParseFloat(freq, 64)
}

var iwgetid = func(intf, flag string) (string, error) {
	out, err := exec.Command("/sbin/iwgetid", intf, "-r", flag).Output()
	return strings.TrimSpace(string(out)), err
}
