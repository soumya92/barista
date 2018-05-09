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

	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
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
	bar.Module
	bar.Clickable

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
	provider       Provider
	scheduler      bar.Scheduler
	outputFunc     base.Value // of func(Weather) bar.Output
	clickHandler   base.Value // of func(Weather, bar.Event)
	currentWeather base.Value // of Weather
}

func defaultOutputFunc(w Weather) bar.Output {
	return outputs.Textf("%.1f℃ %s", w.Temperature.Celsius(), w.Description)
}

// New constructs an instance of the weather module with the provided configuration.
func New(provider Provider) Module {
	m := &module{
		provider:  provider,
		scheduler: base.Schedule().Every(10 * time.Minute),
	}
	// Default output template is just the temperature and conditions.
	m.OutputTemplate(outputs.TextTemplate(`{{.Temperature.C | printf "%.1f"}}℃ {{.Description}}`))
	m.OnClick(nil)
	return m
}

func (m *module) OutputFunc(outputFunc func(Weather) bar.Output) Module {
	m.outputFunc.Set(outputFunc)
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
		f = func(w Weather, e bar.Event) {}
	}
	m.clickHandler.Set(f)
	return m
}

func (m *module) Click(e bar.Event) {
	clickHandler := m.clickHandler.Get().(func(Weather, bar.Event))
	if w := m.currentWeather.Get(); w != nil {
		clickHandler(w.(Weather), e)
	}
}

func (m *module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *module) worker(ch base.Channel) {
	weather, err := m.provider.GetWeather()
	outputFunc := m.outputFunc.Get().(func(Weather) bar.Output)
	sOutputFunc := m.outputFunc.Subscribe()
	for {
		if ch.Error(err) {
			return
		}
		if weather != nil {
			m.currentWeather.Set(*weather)
			ch.Output(outputFunc(*weather))
		}
		select {
		case <-sOutputFunc.Tick():
			outputFunc = m.outputFunc.Get().(func(Weather) bar.Output)
		case <-m.scheduler.Tick():
			weather, err = m.provider.GetWeather()
		}
	}
}
