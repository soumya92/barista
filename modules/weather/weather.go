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
package weather // import "barista.run/modules/weather"

import (
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"
	"barista.run/timing"

	"github.com/martinlindhe/unit"
)

// Weather represents the current weather conditions.
type Weather struct {
	Location    string
	Condition   Condition
	Description string
	Temperature unit.Temperature
	Humidity    float64
	Pressure    unit.Pressure
	Wind        Wind
	CloudCover  float64
	Sunrise     time.Time
	Sunset      time.Time
	Updated     time.Time
	Attribution string
}

// Wind stores the wind speed and direction together.
type Wind struct {
	unit.Speed
	Direction
}

// Condition represents a weather condition.
type Condition int

// Possible weather conditions
const (
	ConditionUnknown Condition = iota
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

// Direction represents a compass direction stored as degrees.
type Direction int

// Provider is an interface for weather providers,
// implemented by the various provider packages.
type Provider interface {
	GetWeather() (Weather, error)
}

// Module represents a bar.Module that displays weather information.
type Module struct {
	provider       Provider
	scheduler      timing.Scheduler
	outputFunc     value.Value // of func(Weather) bar.Output
	currentWeather value.Value // of Weather
}

// New constructs an instance of the weather module with the provided configuration.
func New(provider Provider) *Module {
	m := &Module{
		provider:  provider,
		scheduler: timing.NewScheduler(),
	}
	l.Register(m, "outputFunc", "clickHandler", "currentWeather", "scheduler")
	// Default output is just the temperature and conditions.
	m.Output(func(w Weather) bar.Output {
		return outputs.Textf("%.1f℃ %s (%s)",
			w.Temperature.Celsius(), w.Description, w.Attribution)
	})
	m.RefreshInterval(10 * time.Minute)
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Weather) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval configures the polling frequency.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	weather, err := m.provider.GetWeather()
	outputFunc := m.outputFunc.Get().(func(Weather) bar.Output)
	nextOutputFunc := m.outputFunc.Next()
	for {
		if s.Error(err) {
			return
		}
		m.currentWeather.Set(weather)
		s.Output(outputFunc(weather))
		select {
		case <-nextOutputFunc:
			nextOutputFunc = m.outputFunc.Next()
			outputFunc = m.outputFunc.Get().(func(Weather) bar.Output)
		case <-m.scheduler.Tick():
			weather, err = m.provider.GetWeather()
		}
	}
}
