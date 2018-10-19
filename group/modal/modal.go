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

// Package modal provides a group with various "modes", each mode identified by
// a string key and containing multiple modules. It switches between the modes
// using a control similar to the workspace switcher.
//
// When adding modules to a mode, certain modules can be marked as "summary"
// modules, to be displayed when no mode is active. When a mode is active, only
// the modules associated with it are displayed.
//
// For example, if a modal group is constructed with the following sets, where
// uppercase letters indicate summary modules:
//
//   - "A" => "A0", "a1", "a2", "a3"
//   - "B" => "b0", "B1", "B2"
//   - "C" => "c0", "c1", "c2"
//
// Then by default the modules displayed will be ["A0", "B1", "B2"].
// Activating "A" will replace that with ["A0", "a1", "a2", "a3"],
// "B" will show ["b0", "B1", "B2"], and "C" will show ["c0", "c1", "c2"].
package modal // import "barista.run/group/modal"

import (
	"sync"
	"sync/atomic"

	"barista.run/bar"
	"barista.run/base/click"
	"barista.run/base/notifier"
	"barista.run/colors"
	"barista.run/group"
	l "barista.run/logging"
	"barista.run/outputs"
)

const (
	showWhenSummary int = 1 << iota
	showWhenDetail
)

// Controller provides an interface to control a modal group.
type Controller interface {
	// Modes returns all the modes in this modal group.
	Modes() []string
	// Current returns the currently active mode, or an empty string if no mode
	// is active.
	Current() string
	// Activate activates the given mode.
	Activate(string)
	// Toggle toggles between the given mode and no active mode.
	Toggle(string)
	// Reset clears the active mode.
	Reset()
	// SetOutput sets the output segment for a given mode. The default output
	// is a plain text segment with the mode name.
	SetOutput(string, *bar.Segment)
}

// grouper implements a modal grouper.
type grouper struct {
	current   atomic.Value // of string
	showWhen  map[int]int
	mode      map[int]string
	modeNames []string
	output    map[string]*bar.Segment

	sync.Mutex
	notifyCh <-chan struct{}
	notifyFn func()
}

// Modal represents a partially constructed modal group. Modes and modules can
// only be added to a Modal before it is finalised, and can only be added to
// the bar after it is finalised.
type Modal struct {
	modes     map[string]*Mode
	modeNames []string
}

// Mode represents a mode added to an existing modal group. It provides methods
// to add additional outputs and optionally set the default output.
type Mode struct {
	output   *bar.Segment
	modules  []bar.Module
	showWhen map[int]int
}

// New creates a new modal group.
func New() *Modal {
	return &Modal{modes: map[string]*Mode{}}
}

// Mode creates a new mode.
func (m *Modal) Mode(label string) *Mode {
	md := &Mode{
		output:   outputs.Text(label),
		showWhen: map[int]int{},
	}
	m.modeNames = append(m.modeNames, label)
	m.modes[label] = md
	return md
}

// Summary adds a summary module to the mode. Summary modules are shown when the
// no mode is active.
func (m *Mode) Summary(modules ...bar.Module) *Mode {
	return m.add(showWhenSummary, modules)
}

// Detail adds a detail module to a mode. Modules added here are only shown when
// this mode is active.
func (m *Mode) Detail(modules ...bar.Module) *Mode {
	return m.add(showWhenDetail, modules)
}

// Add adds a module in both summary and detail modes. Modules added here are
// shown both when the current mode is active and when no mode is active. They
// are only hidden when a different mode is active.
func (m *Mode) Add(modules ...bar.Module) *Mode {
	return m.add(showWhenSummary|showWhenDetail, modules)
}

// add adds modules with the given visibility flags.
func (m *Mode) add(flags int, modules []bar.Module) *Mode {
	for _, mod := range modules {
		m.showWhen[len(m.modules)] = flags
		m.modules = append(m.modules, mod)
	}
	return m
}

// SetOutput sets the output shown in the mode switcher. The default output
// is just the name of the mode.
func (m *Mode) SetOutput(s *bar.Segment) *Mode {
	m.output = s
	return m
}

// Build constructs the modal group, and returns a linked controller.
func (m *Modal) Build() (bar.Module, Controller) {
	g := &grouper{
		modeNames: m.modeNames,
		showWhen:  map[int]int{},
		mode:      map[int]string{},
		output:    map[string]*bar.Segment{},
	}
	modules := []bar.Module{}
	for _, modeName := range m.modeNames {
		mode := m.modes[modeName]
		start := len(modules)
		g.output[modeName] = mode.output
		modules = append(modules, mode.modules...)
		for k, v := range mode.showWhen {
			g.showWhen[k+start] = v
		}
		for i := start; i < len(modules); i++ {
			g.mode[i] = modeName
		}
	}
	g.current.Store("")
	g.notifyFn, g.notifyCh = notifier.New()
	return group.New(g, modules...), g
}

func (g *grouper) Visible(idx int) bool {
	switch g.Current() {
	case g.mode[idx]:
		return g.showWhen[idx]&showWhenDetail > 0
	case "":
		return g.showWhen[idx]&showWhenSummary > 0
	default:
		return false
	}
}

func (g *grouper) Buttons() (start, end bar.Output) {
	out := outputs.Group().Glue()
	for _, mode := range g.modeNames {
		s := g.output[mode]
		if s == nil {
			continue
		}
		colorKey := "inactive"
		if g.Current() == mode {
			colorKey = "focused"
		}
		mode := mode
		out.Append(s.Background(colors.Scheme(colorKey + "_workspace_bg")).
			Color(colors.Scheme(colorKey + "_workspace_text")).
			Border(colors.Scheme(colorKey + "_workspace_border")).
			OnClick(click.Left(func() { g.Toggle(mode) })))
	}
	return nil, out
}

func (g *grouper) Signal() <-chan struct{} {
	return g.notifyCh
}

func (g *grouper) Modes() []string {
	return g.modeNames
}

func (g *grouper) Current() string {
	return g.current.Load().(string)
}

func (g *grouper) Activate(mode string) {
	g.set(mode)
}

func (g *grouper) Toggle(mode string) {
	if g.Current() == mode {
		g.set("")
	} else {
		g.set(mode)
	}
}

func (g *grouper) Reset() {
	g.set("")
}

func (g *grouper) set(mode string) {
	g.Lock()
	defer g.Unlock()
	if g.Current() == mode {
		return
	}
	g.current.Store(mode)
	l.Fine("%s switched to '%s'", l.ID(g), mode)
	g.notifyFn()
}

func (g *grouper) SetOutput(mode string, segment *bar.Segment) {
	g.Lock()
	defer g.Unlock()
	if segment != nil {
		segment = segment.Clone()
	}
	g.output[mode] = segment
	g.notifyFn()
}
