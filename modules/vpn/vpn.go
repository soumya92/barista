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

	"github.com/google/barista/bar"
	"github.com/google/barista/bar/outputs"
	"github.com/google/barista/modules/base"

	"github.com/vishvananda/netlink"
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

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(State) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(s State) *bar.Output {
		return template(s)
	})
}

// Interface sets the name of the interface to use for checking vpn state.
type Interface string

func (i Interface) apply(m *module) {
	m.intf = string(i)
}

type module struct {
	*base.Base
	outputFunc func(State) *bar.Output
	intf       string
	lastFlags  uint32
}

// New constructs an instance of the wlan module with the provided configuration.
func New(config ...Config) base.Module {
	m := &module{
		Base: base.New(),
		// Default interface for openvpn. Override using Interface(...)
		intf: "tun0",
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just 'VPN' when connected.
		defTpl := outputs.TextTemplate("{{if .Connected}}VPN{{end}}")
		OutputTemplate(defTpl).apply(m)
	}
	// Worker goroutine to watch for vpn state changes.
	m.SetWorker(m.loop)
	return m
}

func (m *module) loop() error {
	// Initial state.
	state := Disconnected
	if link, err := netlink.LinkByName(m.intf); err == nil {
		if link.Attrs().Flags&net.FlagUp == net.FlagUp {
			state = Connected
		} else {
			state = Waiting
		}
	}
	m.Output(m.outputFunc(state))

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
			state := Disconnected
			if newFlags&syscall.IFF_RUNNING == syscall.IFF_RUNNING {
				state = Connected
			} else if newFlags&syscall.IFF_UP == syscall.IFF_UP {
				state = Waiting
			}
			m.Output(m.outputFunc(state))
		}
	}
	return nil
}
