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
)

// State represents the network state.
type State struct {
	netlink.Link
}

// Connected returns true if a network is up.
func (s State) Connected() bool {
	return s.State == netlink.Up
}

// Disconnected returns true if no networks are up.
func (s State) Disconnected() bool {
	return !s.Disabled() && s.State < netlink.Dormant
}

// Disabled returns true if no links are present.
func (s State) Disabled() bool {
	return s.State <= netlink.NotPresent
}

// Module represents a netinfo bar module.
type Module struct {
	base.SimpleClickHandler
	subscriber func() netlink.Subscription
	outputFunc base.Value // of func(State) bar.Output
}

// netWithSubscriber constructs a netinfo module using the given
// subscriber function.
func newWithSubscriber(subscriber func() netlink.Subscription) *Module {
	m := &Module{subscriber: subscriber}
	l.Register(m, "outputFunc")
	// Default output template is the name of the connected interface.
	m.Template("{{if .Connected}}{{.Name}}{{end}}")
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

// Template configures a module to display the output of a template.
func (m *Module) Template(template string) *Module {
	base.Template(template, m.Output)
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
		case <-m.outputFunc.Update():
			outputFunc = m.outputFunc.Get().(func(State) bar.Output)
		}
		s.Output(outputFunc(state))
	}
}
