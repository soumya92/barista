// Copyright 2018 Google Inc.
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

// Package netinfo provides an i3bar module for network information.
package netinfo

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/watchers/netlink"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// State represents the network state.
type State struct {
	netlink.Link
}

// Connecting returns true if a connection is in progress.
func (s State) Connecting() bool {
	return s.State == netlink.Dormant
}

// Connected returns true if connected to a network.
func (s State) Connected() bool {
	return s.State == netlink.Up
}

// Enabled returns true if a network interface is enabled.
func (s State) Enabled() bool {
	return s.State > netlink.NotPresent
}

// Module represents a netinfo bar module.
type Module struct {
	subscriber func() netlink.Subscription
	outputFunc base.Value // of func(State) bar.Output
}

// netWithSubscriber constructs a netinfo module using the given
// subscriber function.
func newWithSubscriber(subscriber func() netlink.Subscription) *Module {
	m := &Module{subscriber: subscriber}
	l.Register(m, "outputFunc")
	// Default output is the name of the connected interface.
	m.Output(func(s State) bar.Output {
		if s.Connected() {
			return outputs.Text(s.Name)
		}
		return nil
	})
	return m
}

// New constructs a netinfo module that scans all interfaces.
func New() *Module {
	m := newWithSubscriber(netlink.Any)
	l.Label(m, "*")
	return m
}

// Interface constructs an instance of the netinfo module
// restricted to the specified interface.
func Interface(iface string) *Module {
	m := newWithSubscriber(func() netlink.Subscription {
		return netlink.ByName(iface)
	})
	l.Label(m, iface)
	return m
}

// Prefix constructs an instance of the netinfo module restricted
// to interfaces with the given prefix.
func Prefix(prefix string) *Module {
	m := newWithSubscriber(func() netlink.Subscription {
		return netlink.WithPrefix(prefix)
	})
	l.Labelf(m, "%s*", prefix)
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(State) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	var state State
	outputFunc := m.outputFunc.Get().(func(State) bar.Output)
	linkCh := m.subscriber()
	defer linkCh.Unsubscribe()

	for {
		select {
		case update := <-linkCh:
			state = State{update}
		case <-m.outputFunc.Next():
			outputFunc = m.outputFunc.Get().(func(State) bar.Output)
		}
		s.Output(outputFunc(state))
	}
}
