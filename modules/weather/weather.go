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
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
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

// Module represents a bar.Module that displays weather information.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to get the current weather.
type Module struct {
	provider       Provider
	scheduler      timing.Scheduler
	outputFunc     base.Value // of func(Weather) bar.Output
	clickHandler   base.Value // of func(Weather, bar.Event)
	currentWeather base.Value // of Weather
}

func defaultOutputFunc(w Weather) bar.Output {
	return outputs.Textf("%.1f℃ %s", w.Temperature.Celsius(), w.Description)
}

// New constructs an instance of the weather module with the provided configuration.
func New(provider Provider) *Module {
	m := &Module{
		provider:  provider,
		scheduler: timing.NewScheduler(),
	}
	l.Register(m, "outputFunc", "clickHandler", "currentWeather", "scheduler")
	// Default output template is just the temperature and conditions.
	m.OutputTemplate(outputs.TextTemplate(`{{.Temperature.C | printf "%.1f"}}℃ {{.Description}}`))
	m.RefreshInterval(10 * time.Minute)
	m.OnClick(nil)
	return m
}

// OutputFunc configures a module to display the output of a user-defined function.
func (m *Module) OutputFunc(outputFunc func(Weather) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *Module) OutputTemplate(template func(interface{}) bar.Output) *Module {
	return m.OutputFunc(func(w Weather) bar.Output {
		return template(w)
	})
}

// RefreshInterval configures the polling frequency.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

// OnClick sets a click handler for the module.
func (m *Module) OnClick(f func(Weather, bar.Event)) *Module {
	if f == nil {
		f = func(w Weather, e bar.Event) {}
	}
	m.clickHandler.Set(f)
	return m
}

// Click handles click events on the module's output.
func (m *Module) Click(e bar.Event) {
	clickHandler := m.clickHandler.Get().(func(Weather, bar.Event))
	if w := m.currentWeather.Get(); w != nil {
		clickHandler(w.(Weather), e)
	}
}

// Stream starts the module.
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *Module) worker(ch base.Channel) {
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
		case <-sOutputFunc:
			outputFunc = m.outputFunc.Get().(func(Weather) bar.Output)
		case <-m.scheduler.Tick():
			weather, err = m.provider.GetWeather()
		}
	}
}
