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

// Package clock displays a clock.
package clock

import (
	"time"

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/scheduler"
)

// Module represents a clock bar module. It supports setting the click handler,
// timezone, output format, and granularity.
type Module interface {
	base.SimpleClickHandlerModule

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(time.Time) bar.Output) Module

	// OutputFormat configures a module to display the time in a given format.
	OutputFormat(string) Module

	// Timezone configures the timezone for this clock.
	Timezone(string) Module

	// Granularity configures the granularity at which the module should refresh.
	// For example, if your format does not have seconds, you can set it to time.Minute.
	// The module will always update at the next second, minute, hour, etc.,
	// so you don't need to be concerned about update delays with a large granularity.
	Granularity(time.Duration) Module
}

type module struct {
	base.SimpleClickHandler
	config base.Value
}

type config struct {
	granularity time.Duration
	outputFunc  func(time.Time) bar.Output
	timezone    string
}

func (m *module) getConfig() config {
	return m.config.Get().(config)
}

func defaultOutputFunc(now time.Time) bar.Output {
	return outputs.Text(now.Format("15:04"))
}

// New constructs an instance of the clock module with a default configuration.
func New() Module {
	m := &module{}
	m.config.Set(config{
		timezone:    "",
		granularity: time.Second,
		outputFunc:  defaultOutputFunc,
	})
	return m
}

func (m *module) OutputFunc(outputFunc func(time.Time) bar.Output) Module {
	c := m.getConfig()
	c.outputFunc = outputFunc
	m.config.Set(c)
	return m
}

func (m *module) OutputFormat(format string) Module {
	return m.OutputFunc(func(now time.Time) bar.Output {
		return outputs.Text(now.Format(format))
	})
}

func (m *module) Timezone(timezone string) Module {
	c := m.getConfig()
	c.timezone = timezone
	m.config.Set(c)
	return m
}

func (m *module) Granularity(granularity time.Duration) Module {
	c := m.getConfig()
	c.granularity = granularity
	m.config.Set(c)
	return m
}

func (m *module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *module) worker(ch base.Channel) {
	sch := barista.Schedule()
	cfg := m.getConfig()
	tz := time.Local
	prevTz := ""
	sCfg := m.config.Subscribe()
	for {
		if cfg.timezone != prevTz {
			var err error
			tz, err = time.LoadLocation(cfg.timezone)
			if ch.Error(err) {
				return
			}
			prevTz = cfg.timezone
		}

		now := scheduler.Now()
		ch.Output(cfg.outputFunc(now.In(tz)))
		next := now.Add(cfg.granularity).Truncate(cfg.granularity)

		select {
		case <-sch.At(next).Tick():
		case <-sCfg.Tick():
			cfg = m.getConfig()
		}
	}
}
