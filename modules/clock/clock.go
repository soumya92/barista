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
	"strings"
	"time"

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/scheduler"
)

// Module represents a clock bar module. It supports setting the click handler,
// timezone, output format, and granularity.
type Module struct {
	base.SimpleClickHandler
	config base.Value
}

type config struct {
	granularity time.Duration
	outputFunc  func(time.Time) bar.Output
	timezone    *time.Location
}

func (m *Module) getConfig() config {
	return m.config.Get().(config)
}

func defaultOutputFunc(now time.Time) bar.Output {
	return outputs.Text(now.Format("15:04"))
}

// Zone constructs a clock module for the given timezone.
func Zone(timezone *time.Location) *Module {
	m := &Module{}
	m.config.Set(config{
		timezone:    timezone,
		granularity: time.Minute,
		outputFunc:  defaultOutputFunc,
	})
	return m
}

// Local constructs a clock module for the current machine's timezone.
func Local() *Module {
	return Zone(time.Local)
}

// ZoneByName constructs a clock module for the given zone name,
// (e.g. "America/Los_Angeles"), and returns any errors.
func ZoneByName(name string) (*Module, error) {
	tz, err := time.LoadLocation(name)
	if err != nil {
		return nil, err
	}
	return Zone(tz), nil
}

// OutputFunc configures a module to display the output of a user-defined function.
//
// The first argument configures the granularity at which the module should refresh.
// For example, if the format does not have seconds, it should be time.Minute.
//
// The module will always update at the next second, minute, hour, etc.,
// so large granularities will not negatively affect the output.
func (m *Module) OutputFunc(
	granularity time.Duration,
	outputFunc func(time.Time) bar.Output,
) *Module {
	c := m.getConfig()
	c.granularity = granularity
	c.outputFunc = outputFunc
	m.config.Set(c)
	return m
}

// OutputFormat configures a module to display the time in a given format.
func (m *Module) OutputFormat(format string) *Module {
	granularity := time.Hour
	switch {
	case strings.Contains(format, ".000"):
		granularity = time.Millisecond
	case strings.Contains(format, ".00"):
		granularity = 10 * time.Millisecond
	case strings.Contains(format, ".0"):
		granularity = 100 * time.Millisecond
	case strings.Contains(format, "05"):
		granularity = time.Second
	case strings.Contains(format, "04"):
		granularity = time.Minute
	}
	return m.OutputFunc(granularity, func(now time.Time) bar.Output {
		return outputs.Text(now.Format(format))
	})
}

// Timezone configures the timezone for this clock.
func (m *Module) Timezone(timezone *time.Location) *Module {
	c := m.getConfig()
	c.timezone = timezone
	m.config.Set(c)
	return m
}

// Stream starts the module.
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *Module) worker(ch base.Channel) {
	sch := barista.NewScheduler()
	cfg := m.getConfig()
	sCfg := m.config.Subscribe()
	for {
		now := scheduler.Now()
		ch.Output(cfg.outputFunc(now.In(cfg.timezone)))
		next := now.Add(cfg.granularity).Truncate(cfg.granularity)

		select {
		case <-sch.At(next).Tick():
		case <-sCfg:
			cfg = m.getConfig()
		}
	}
}
