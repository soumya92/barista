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
	"syscall"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/bar/outputs"
	"github.com/soumya92/barista/modules/base"

	"github.com/vishvananda/netlink"
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

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(Info) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(i Info) *bar.Output {
		return template(i)
	})
}

// Interface sets the name of the interface to display the status for.
type Interface string

func (i Interface) apply(m *module) {
	m.intf = string(i)
}

type module struct {
	*base.Base
	outputFunc func(Info) *bar.Output
	intf       string
	info       Info
	lastFlags  uint32
}

// New constructs an instance of the wlan module with the provided configuration.
func New(config ...Config) base.Module {
	m := &module{
		Base: base.New(),
		// Default interface for goobuntu laptops. Override using Interface(...)
		intf: "wlan0",
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just the SSID when connected.
		defTpl := outputs.TextTemplate("{{if .Connected}}{{.SSID}}{{end}}")
		OutputTemplate(defTpl).apply(m)
	}
	// Worker goroutine to watch for changes to the interface state.
	m.SetWorker(m.loop)
	return m
}

func (m *module) loop() error {
	// Initial state.
	link, err := netlink.LinkByName(m.intf)
	if err != nil {
		return err
	}
	m.info = Info{}
	if link.Attrs().Flags&net.FlagUp == net.FlagUp {
		if err := m.getWifiInfo(); err != nil {
			return err
		}
		if m.info.SSID == "" {
			m.info.State = Disconnected
		}
	}
	m.refresh()

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
		if m.lastFlags&syscall.IFF_UP != newFlags&syscall.IFF_UP {
			shouldUpdate = true
		}
		if m.lastFlags&syscall.IFF_RUNNING != newFlags&syscall.IFF_RUNNING {
			shouldUpdate = true
		}
		if shouldUpdate {
			m.lastFlags = newFlags
			m.info = Info{}
			if newFlags&syscall.IFF_UP != syscall.IFF_UP {
				m.info.State = Disabled
			} else if newFlags&syscall.IFF_RUNNING != syscall.IFF_RUNNING {
				m.info.State = Disconnected
			} else {
				if err := m.getWifiInfo(); err != nil {
					return err
				}
			}
			m.refresh()
		}
	}
	return nil
}

func (m *module) refresh() {
	m.Output(m.outputFunc(m.info))
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
