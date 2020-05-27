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
package clock // import "barista.run/modules/clock"

import (
	"strings"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/base/watchers/localtz"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"
)

// Module represents a clock bar module. It supports setting the click handler,
// timezone, output format, and granularity.
type Module struct {
	config value.Value
}

type config struct {
	granularity time.Duration
	outputFunc  func(time.Time) bar.Output
	timezone    *time.Location
}

func (m *Module) getConfig() config {
	return m.config.Get().(config)
}

func defaultOutput(now time.Time) bar.Output {
	return outputs.Text(now.Format("15:04"))
}

// Zone constructs a clock module for the given timezone.
func Zone(timezone *time.Location) *Module {
	m := &Module{}
	l.Register(m, "config")
	m.config.Set(config{
		timezone:    timezone,
		granularity: time.Minute,
		outputFunc:  defaultOutput,
	})
	return m
}

// Local constructs a clock module for the current machine's timezone.
func Local() *Module {
	return Zone(nil)
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

// Output configures a module to display the output of a user-defined function.
//
// The first argument configures the granularity at which the module should refresh.
// For example, if the format does not have seconds, it should be time.Minute.
//
// The module will always update at the next second, minute, hour, etc.,
// so large granularities will not negatively affect the output due to drift.
//
// Additionally, the module will automatically update when discontinuous change of
// clock occurs, due to e.g. system suspend or clock adjustments.
func (m *Module) Output(
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
	case strings.Contains(format, ".000"), strings.Contains(format, ".999"):
		granularity = time.Millisecond
	case strings.Contains(format, ".00"), strings.Contains(format, ".99"):
		granularity = 10 * time.Millisecond
	case strings.Contains(format, ".0"), strings.Contains(format, ".9"):
		granularity = 100 * time.Millisecond
	case strings.Contains(format, "05"), strings.Contains(format, "_5"):
		granularity = time.Second
	case strings.Contains(format, "04"), strings.Contains(format, "_4"):
		granularity = time.Minute
	}
	return m.Output(granularity, func(now time.Time) bar.Output {
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
func (m *Module) Stream(s bar.Sink) {
	sch, err := timing.NewRealtimeScheduler()
	if s.Error(err) {
		return
	}
	defer sch.Close()
	l.Attach(m, sch, ".scheduler")

	cfg := m.getConfig()
	nextCfg, done := m.config.Subscribe()
	defer done()

	var tzChange <-chan struct{}

	sch.EveryAlign(cfg.granularity, time.Duration(0))

	for {
		now := timing.Now()

		if cfg.timezone == nil {
			if tzChange == nil {
				tzChange = localtz.Next()
			}
		} else {
			now = now.In(cfg.timezone)
			tzChange = nil
		}
		s.Output(cfg.outputFunc(now))

		select {
		case <-sch.C:
		case <-tzChange:
			tzChange = nil
		case <-nextCfg:
			cfg = m.getConfig()
			sch.EveryAlign(cfg.granularity, time.Duration(0))
		}
	}
}
