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

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(time.Time) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputFormat configures a module to display the time in a given format.
func OutputFormat(format string) Config {
	return OutputFunc(func(now time.Time) *bar.Output {
		return outputs.Text(now.Format(format))
	})
}

// Timezone configures the timezone for this clock.
type Timezone string

func (t Timezone) apply(m *module) {
	var err error
	m.timezone, err = time.LoadLocation(string(t))
	m.Error(err)
}

// Granularity configures the granularity at which the module should refresh.
// For example, if your format does not have seconds, you can set it to time.Minute.
// The module will always update at the next second, minute, hour, etc.,
// so you don't need to be concerned about update delays with a large granularity.
type Granularity time.Duration

func (g Granularity) apply(m *module) {
	m.granularity = time.Duration(g)
}

type module struct {
	*base.Base
	granularity time.Duration
	outputFunc  func(time.Time) *bar.Output
	timezone    *time.Location
}

// New constructs an instance of the clock module with the provided configuration.
func New(config ...Config) base.WithClickHandler {
	m := &module{
		Base: base.New(),
		// Default granularity is 1 second, to avoid confusing users.
		granularity: time.Second,
		// Default to machine's timezone.
		timezone: time.Local,
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		OutputFormat("15:04").apply(m)
		Granularity(time.Minute).apply(m)
	}
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
