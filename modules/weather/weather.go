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

// Package weather provides an i3bar module that displays weather info.
package weather

import (
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
	"github.com/soumya92/barista/outputs"
)

// Weather represents the current weather conditions.
type Weather struct {
	Location    string
	Condition   Condition
	Description string
	Temperature Temperature
	Humidity    float64
	Pressure    Pressure
	Wind        Wind
	CloudCover  float64
	Sunrise     time.Time
	Sunset      time.Time
	Updated     time.Time
	Attribution string
}

// Wind stores the wind speed and direction together.
type Wind struct {
	Speed
	Direction
}

// Condition represents a weather condition.
type Condition int

// Possible weather conditions
const (
	ConditionUnknown = iota
	Thunderstorm
	Drizzle
	Rain
	Snow
	Sleet
	Mist
	Smoke
	Whirls
	Haze
	Fog
	Clear
	Cloudy
	PartlyCloudy
	Overcast
	Tornado
	TropicalStorm
	Hurricane
	Cold
	Hot
	Windy
	Hail
)

// Temperature provides unit conversions for temperature,
// and stores the temperature in kelvin.
type Temperature float64

// Pressure provides unit conversions for pressure,
// and stores the temperature in millibar.
type Pressure float64

// Speed provides unit conversions for speed,
// and stores the speed in meters per second.
type Speed float64

// Direction represents a compass direction stored as degrees.
type Direction int

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(Weather) *bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) *bar.Output) Config {
	return OutputFunc(func(w Weather) *bar.Output {
		return template(w)
	})
}

// RefreshInterval configures the polling frequency.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
}

// Provider is an interface for weather providers,
type Provider interface {
	GetWeather() (*Weather, error)
}

// Module is the public interface for a weather module.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to get the current weather.
type Module interface {
	base.Module
	OnClick(func(Weather, bar.Event))
}

type module struct {
	*base.Base
	provider        Provider
	refreshInterval time.Duration
	outputFunc      func(Weather) *bar.Output
	// cache last weather info for click handler.
	lastWeather Weather
}

// New constructs an instance of the weather module with the provided configuration.
func New(provider Provider, config ...Config) Module {
	m := &module{
		Base: base.New(),
		// Provider is required, so it's a param instead of being an optional Config.
		provider: provider,
		// Default is to refresh every 10 minutes
		refreshInterval: 10 * time.Minute,
	}
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just the temperature and conditions.
		defTpl := outputs.TextTemplate(`{{.Temperature.C}}℃ {{.Description}}`)
		OutputTemplate(defTpl).apply(m)
	}
	// Worker goroutine to update weather at an interval.
	m.OnUpdate(m.update)
	m.UpdateEvery(m.refreshInterval)
	return m
}

// OnClick sets a click handler for the module.
func (m *module) OnClick(f func(Weather, bar.Event)) {
	if f == nil {
		m.Base.OnClick(nil)
		return
	}
	m.Base.OnClick(func(e bar.Event) {
		f(m.lastWeather, e)
	})
}

func (m *module) update() {
	weather, err := m.provider.GetWeather()
	if m.Error(err) || weather == nil {
		// nil weather means unchanged.
		return
	}
	m.lastWeather = *weather
	m.Output(m.outputFunc(m.lastWeather))
}
