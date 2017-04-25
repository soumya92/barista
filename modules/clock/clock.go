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
	m.timezone = string(t)
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
	timezone    string
}

// New constructs an instance of the clock module with the provided configuration.
func New(config ...Config) base.Module {
	m := &module{
		Base: base.New(),
		// Default granularity is 1 second, to avoid confusing users.
		granularity: time.Second,
		// Default to machine's timezone.
		timezone: "Local",
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
	// Worker goroutine to update load average at a fixed interval.
	m.SetWorker(m.update)
	return m
}

func (m *module) update() error {
	zone, err := time.LoadLocation(m.timezone)
	if err != nil {
		return err
	}
	for {
		t := time.Now()
		m.Output(m.outputFunc(t.In(zone)))
		// Sleep until the next time the granularity unit changes..
		sleepDuration := t.Add(m.granularity).Truncate(m.granularity).Sub(t)
		time.Sleep(sleepDuration)
	}
}
