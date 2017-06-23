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
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/base/scheduler"
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

// Provider is an interface for weather providers,
// implemented by the various provider packages.
type Provider interface {
	GetWeather() (*Weather, error)
}

// Module is the public interface for a weather module.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to get the current weather.
type Module interface {
	base.Module

	// RefreshInterval configures the polling frequency.
	RefreshInterval(time.Duration) Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Weather) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OnClick sets a click handler for the module.
	OnClick(func(Weather, bar.Event)) Module
}

type module struct {
	*base.Base
	provider   Provider
	scheduler  scheduler.Scheduler
	outputFunc func(Weather) bar.Output
	// cache last weather info for click handler.
	lastWeather Weather
}

// New constructs an instance of the weather module with the provided configuration.
func New(provider Provider) Module {
	m := &module{
		Base:     base.New(),
		provider: provider,
	}
	// Default is to refresh every 10 minutes
	m.scheduler = scheduler.Do(m.Update).Every(10 * time.Minute)
	// Default output template is just the temperature and conditions.
	m.OutputTemplate(outputs.TextTemplate(`{{.Temperature.C}}℃ {{.Description}}`))
	// Update weather when asked.
	m.OnUpdate(m.update)
	return m
}

func (m *module) OutputFunc(outputFunc func(Weather) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(w Weather) bar.Output {
		return template(w)
	})
}

func (m *module) RefreshInterval(interval time.Duration) Module {
	m.scheduler.Every(interval)
	return m
}

func (m *module) OnClick(f func(Weather, bar.Event)) Module {
	if f == nil {
		m.Base.OnClick(nil)
		return m
	}
	m.Base.OnClick(func(e bar.Event) {
		f(m.lastWeather, e)
	})
	return m
}

func (m *module) update() {
	weather, err := m.provider.GetWeather()
	if m.Error(err) {
		return
	}
	if weather != nil {
		// nil weather means unchanged.
		m.lastWeather = *weather
	}
	m.Output(m.outputFunc(m.lastWeather))
}
