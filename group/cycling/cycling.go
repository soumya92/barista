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

// Package cycling provides a group that continuously cycles between
// all modules at a fixed interval.
package cycling

import (
	"sync"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/group"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/notifier"
	"github.com/soumya92/barista/timing"
)

// Controller provides an interface to control a collapsing group.
type Controller interface {
	SetInterval(time.Duration)
}

// grouper implements a cycling grouper.
type grouper struct {
	current   int
	count     int
	scheduler timing.Scheduler

	sync.Mutex
	notifyCh <-chan struct{}
	notifyFn func()
}

// Group returns a new cycling group with the given interval,
// and a linked Controller.
func Group(interval time.Duration, m ...bar.Module) (bar.Module, Controller) {
	g := &grouper{count: len(m), scheduler: timing.NewScheduler()}
	g.scheduler.Every(interval)
	g.notifyFn, g.notifyCh = notifier.New()
	go g.cycle()
	return group.New(g, m...), g
}

func (g *grouper) Visible(idx int) bool { return g.current == idx }

func (g *grouper) Buttons() (start, end bar.Output) { return nil, nil }

func (g *grouper) ClickStart(bar.Event) {}
func (g *grouper) ClickEnd(bar.Event)   {}

func (g *grouper) Signal() <-chan struct{} { return g.notifyCh }

func (g *grouper) cycle() {
	for range g.scheduler.Tick() {
		g.Lock()
		l.Fine("%s %d++", l.ID(g), g.current)
		g.current = (g.current + 1) % g.count
		g.Unlock()
		g.notifyFn()
	}
}

func (g *grouper) SetInterval(interval time.Duration) {
	g.scheduler.Every(interval)
}
