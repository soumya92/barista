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

// Package vpn provides an i3bar module for openvpn information.
package vpn

import (
	"net"
	"syscall"

	"github.com/vishvananda/netlink"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
)

// State represents the vpn state.
type State int

// Connected returns true if the VPN is connected.
func (s State) Connected() bool {
	return s == Connected
}

// Disconnected returns true if the VPN is off.
func (s State) Disconnected() bool {
	return s == Disconnected
}

// Valid states for the vpn
const (
	Connected State = iota
	Waiting
	Disconnected
)

// Module represents a VPN bar module.
type Module interface {
	base.WithClickHandler

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(State) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module
}

type module struct {
	*base.Base
	outputFunc func(State) bar.Output
	intf       string
	state      State
	lastFlags  uint32
}

// New constructs an instance of the VPN module for the specified interface.
func New(iface string) Module {
	m := &module{
		Base: base.New(),
		intf: iface,
	}
	// Default output template that's just 'VPN' when connected.
	m.OutputTemplate(outputs.TextTemplate("{{if .Connected}}VPN{{end}}"))
	m.OnUpdate(m.update)
	return m
}

// DefaultInterface constructs an instance of the VPN module for "tun0",
// the usual interface for VPNs.
func DefaultInterface() Module {
	return New("tun0")
}

func (m *module) OutputFunc(outputFunc func(State) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(s State) bar.Output {
		return template(s)
	})
}

func (m *module) Stream() <-chan bar.Output {
	go m.worker()
	return m.Base.Stream()
}

func (m *module) worker() {
	// Initial state.
	m.state = Disconnected
	if link, err := netlink.LinkByName(m.intf); err == nil {
		if link.Attrs().Flags&net.FlagUp == net.FlagUp {
			m.state = Connected
		} else {
			m.state = Waiting
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
		if m.lastFlags&syscall.IFF_UP != newFlags&syscall.IFF_UP {
			shouldUpdate = true
		}
		if m.lastFlags&syscall.IFF_RUNNING != newFlags&syscall.IFF_RUNNING {
			shouldUpdate = true
		}
		if shouldUpdate {
			m.lastFlags = newFlags
			m.state = Disconnected
			if newFlags&syscall.IFF_RUNNING == syscall.IFF_RUNNING {
				m.state = Connected
			} else if newFlags&syscall.IFF_UP == syscall.IFF_UP {
				m.state = Waiting
			}
			m.Update()
		}
	}
}

func (m *module) update() {
	m.Output(m.outputFunc(m.state))
}
