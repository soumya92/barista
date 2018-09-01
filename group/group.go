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

// Package group provides a module that groups existing modules, and uses
// a provided Grouper to selectively display output from these modules.
package group

import (
	"sync"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/core"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// Grouper controls how a group displays the output from it's modules.
type Grouper interface {
	// Visible returns true if the module at a given index is visible.
	Visible(index int) bool
	// Button returns the bar output for the buttons on either end.
	Buttons() (start, end bar.Output)
}

// Signaller adds an additional source of updates to the group,
// based on changes that do not cause any of the modules to refresh.
type Signaller interface {
	// Signal returns a channel that signals any updates from the grouper.
	// Signals to this channel will cause the group to recalculate output.
	Signal() <-chan struct{}
}

// UpdateListener receives an update whenever a module in the group
// updates its output.
type UpdateListener interface {
	// Updated is called with the index of the module that just updated
	// its output, before the calls to Button(...) or Visble(...)
	Updated(index int)
}

// group is a general-purpose grouped module that can show
// a subset of the wrapped modules, with buttons on either end.
type group struct {
	grouper   Grouper
	moduleSet *core.ModuleSet
}

// New constructs a new group using the given Grouper and modules.
func New(g Grouper, m ...bar.Module) bar.Module {
	grp := &group{g, core.NewModuleSet(m)}
	l.Register(grp, "grouper", "moduleSet")
	return grp
}

// Stream starts the modules and wraps their before sending it to the bar.
func (g *group) Stream(sink bar.Sink) {
	moduleSetCh := g.moduleSet.Stream()
	var signalCh <-chan struct{}
	if sig, ok := g.grouper.(Signaller); ok {
		signalCh = sig.Signal()
	}
	idx := -1
	for {
		out, changed := g.output(idx)
		if changed || idx < 0 {
			sink(out)
		}
		select {
		case <-signalCh:
			idx = -1
			l.Fine("%s updated from grouper signal", l.ID(g))
		case idx = <-moduleSetCh:
			l.Fine("%s updated from #%d", l.ID(g), idx)
			if u, ok := g.grouper.(UpdateListener); ok {
				u.Updated(idx)
			}
		}
	}
}

// output creates the complete output from this Group.
func (g *group) output(moduleIdx int) (o bar.Output, changed bool) {
	if l, ok := g.grouper.(sync.Locker); ok {
		l.Lock()
		defer l.Unlock()
	}
	out := outputs.Group()
	stBtn, eBtn := g.grouper.Buttons()
	out.Append(stBtn)
	for idx, o := range g.moduleSet.LastOutputs() {
		if !g.grouper.Visible(idx) {
			continue
		}
		out.Append(o)
		if idx == moduleIdx {
			changed = true
		}
	}
	out.Append(eBtn)
	return out, changed
}
