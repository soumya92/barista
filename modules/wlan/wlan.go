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
	l "github.com/soumya92/barista/logging"
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
	templateFn := outputs.TextTemplate(template)
	return m.Output(func(i Info) bar.Output {
		return templateFn(i)
	})
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	// Initial state.
	link, err := netlink.LinkByName(m.intf)
	if s.Error(err) {
		return
	}
	var info Info
	if link.Attrs().Flags&net.FlagUp == net.FlagUp {
		info, err = m.getWifiInfo()
		if s.Error(err) {
			return
		}
		if info.SSID == "" {
			info.State = Disconnected
		}
	}
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	s.Output(outputFunc(info))

	// Watch for changes.
	linkCh := make(chan netlink.LinkUpdate)
	done := make(chan struct{})
	defer close(done)
	netlink.LinkSubscribe(linkCh, done)
	var lastFlags uint32

	for {
		select {
		case update := <-linkCh:
			if update.Attrs().Name != m.intf {
				continue
			}
			newFlags := update.IfInfomsg.Flags
			shouldUpdate := false
			if lastFlags&unix.IFF_UP != newFlags&unix.IFF_UP {
				shouldUpdate = true
			}
			if lastFlags&unix.IFF_RUNNING != newFlags&unix.IFF_RUNNING {
				shouldUpdate = true
			}
			if shouldUpdate {
				lastFlags = newFlags
				info = Info{}
				if newFlags&unix.IFF_UP != unix.IFF_UP {
					info.State = Disabled
				} else if newFlags&unix.IFF_RUNNING != unix.IFF_RUNNING {
					info.State = Disconnected
				} else {
					info, err = m.getWifiInfo()
					if s.Error(err) {
						return
					}
				}
				s.Output(outputFunc(info))
			}
		case <-m.outputFunc.Update():
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
			s.Output(outputFunc(info))
		}
	}
}

func (m *Module) getWifiInfo() (info Info, err error) {
	info.State = Connected
	info.SSID, _ = m.iwgetid("-r")
	info.AccessPointMAC, _ = m.iwgetid("-a")
	var ch string
	if ch, err = m.iwgetid("-c"); err == nil {
		info.Channel, err = strconv.Atoi(ch)
		if err != nil {
			return
		}
	}
	var freq string
	if freq, err = m.iwgetid("-f"); err == nil {
		info.Frequency, err = strconv.ParseFloat(freq, 64)
		if err != nil {
			return
		}
	}
	return
}

func (m *Module) iwgetid(flag string) (string, error) {
	out, err := exec.Command("/sbin/iwgetid", m.intf, "-r", flag).Output()
	return strings.TrimSpace(string(out)), err
}
