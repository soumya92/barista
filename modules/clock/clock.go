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

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
	"github.com/soumya92/barista/outputs"
)

// Module represents a clock bar module. It supports setting the click handler,
// timezone, output format, and granularity.
type Module interface {
	base.WithClickHandler
	OutputFunc(func(time.Time) bar.Output) Module
	OutputFormat(string) Module
	Timezone(string) Module
	Granularity(time.Duration) Module
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *module) OutputFunc(outputFunc func(time.Time) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

// OutputFormat configures a module to display the time in a given format.
func (m *module) OutputFormat(format string) Module {
	return m.OutputFunc(func(now time.Time) bar.Output {
		return outputs.Text(now.Format(format))
	})
}

// Timezone configures the timezone for this clock.
func (m *module) Timezone(timezone string) Module {
	var err error
	m.timezone, err = time.LoadLocation(timezone)
	if !m.Error(err) {
		m.Update()
	}
	return m
}

// Granularity configures the granularity at which the module should refresh.
// For example, if your format does not have seconds, you can set it to time.Minute.
// The module will always update at the next second, minute, hour, etc.,
// so you don't need to be concerned about update delays with a large granularity.
func (m *module) Granularity(granularity time.Duration) Module {
	m.granularity = granularity
	m.Update()
	return m
}

type module struct {
	*base.Base
	granularity time.Duration
	outputFunc  func(time.Time) bar.Output
	timezone    *time.Location
}

// New constructs an instance of the clock module with a default configuration.
func New() Module {
	m := &module{
		Base: base.New(),
		// Default granularity is 1 second, to avoid confusing users.
		granularity: time.Second,
		// Default to machine's timezone.
		timezone: time.Local,
	}
	// Default output template
	m.OutputFormat("15:04")
	m.OnUpdate(m.update)
	return m
}

func (m *module) update() {
	if m.timezone == nil {
		return
	}
	now := time.Now()
	m.Output(m.outputFunc(now.In(m.timezone)))
	m.UpdateAt(now.Add(m.granularity).Truncate(m.granularity))
}
