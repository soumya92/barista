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

	"golang.org/x/sys/unix"

	"github.com/vishvananda/netlink"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
)

// Info represents the wireless card status.
type Info struct {
	State          State
	SSID           string
	AccessPointMAC string
	Channel        int
	Frequency      float64
}

// Connected returns true if connected to a wireless network.
func (i Info) Connected() bool {
	return i.State == Connected
}

// Enabled returns true if the wireless card is enabled.
func (i Info) Enabled() bool {
	return i.State != Disabled
}

// State represents the wireless card state.
type State int

// Valid states for the wireless card.
const (
	Connected State = iota
	Disconnected
	Disabled
)

// Module represents a wlan bar module.
type Module interface {
	base.WithClickHandler

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Info) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module
}

type module struct {
	*base.Base
	outputFunc func(Info) bar.Output
	intf       string
	info       Info
	lastFlags  uint32
}

// New constructs an instance of the wlan module for the specified interface.
func New(iface string) Module {
	m := &module{
		Base: base.New(),
		intf: iface,
	}
	// Default output template is just the SSID when connected.
	m.OutputTemplate(outputs.TextTemplate("{{if .Connected}}{{.SSID}}{{end}}"))
	m.OnUpdate(m.update)
	return m
}

func (m *module) Stream() <-chan bar.Output {
	go m.worker()
	return m.Base.Stream()
}

func (m *module) OutputFunc(outputFunc func(Info) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

func (m *module) worker() {
	// Initial state.
	link, err := netlink.LinkByName(m.intf)
	if m.Error(err) {
		return
	}
	m.info = Info{}
	if link.Attrs().Flags&net.FlagUp == net.FlagUp {
		if m.Error(m.getWifiInfo()) {
			return
		}
		if m.info.SSID == "" {
			m.info.State = Disconnected
		}
	}
	m.Update()

	// Watch for changes.
	ch := make(chan netlink.LinkUpdate)
	done := make(chan struct{})
	defer close(done)
	netlink.LinkSubscribe(ch, done)
	for update := range ch {
		if update.Attrs().Name != m.intf {
			continue
		}
		newFlags := update.IfInfomsg.Flags
		shouldUpdate := false
		if m.lastFlags&unix.IFF_UP != newFlags&unix.IFF_UP {
			shouldUpdate = true
		}
		if m.lastFlags&unix.IFF_RUNNING != newFlags&unix.IFF_RUNNING {
			shouldUpdate = true
		}
		if shouldUpdate {
			m.lastFlags = newFlags
			m.info = Info{}
			if newFlags&unix.IFF_UP != unix.IFF_UP {
				m.info.State = Disabled
			} else if newFlags&unix.IFF_RUNNING != unix.IFF_RUNNING {
				m.info.State = Disconnected
			} else {
				if m.Error(m.getWifiInfo()) {
					return
				}
			}
			m.Update()
		}
	}
}

func (m *module) getWifiInfo() error {
	var err error
	m.info.State = Connected
	m.info.SSID, _ = m.iwgetid("-r")
	m.info.AccessPointMAC, _ = m.iwgetid("-a")
	if ch, err := m.iwgetid("-c"); err == nil {
		m.info.Channel, err = strconv.Atoi(ch)
		if err != nil {
			return err
		}
	}
	if freq, err := m.iwgetid("-f"); err == nil {
		m.info.Frequency, err = strconv.ParseFloat(freq, 64)
		if err != nil {
			return err
		}
	}
	return err
}

func (m *module) iwgetid(flag string) (string, error) {
	out, err := exec.Command("/sbin/iwgetid", m.intf, "-r", flag).Output()
	return strings.TrimSpace(string(out)), err
}

func (m *module) update() {
	m.Output(m.outputFunc(m.info))
}
