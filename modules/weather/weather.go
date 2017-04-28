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
	City        string
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
	ConditionThunderstorm
	ConditionDrizzle
	ConditionRain
	ConditionSnow
	ConditionSleet
	ConditionMist
	ConditionSmoke
	ConditionWhirls
	ConditionHaze
	ConditionFog
	ConditionClear
	ConditionCloudy
	ConditionOvercast
	ConditionTornado
	ConditionTropicalStorm
	ConditionHurricane
	ConditionCold
	ConditionHot
	ConditionWindy
	ConditionHail
)

// Temperature provides unit conversions for temperature,
// and stores the temperature in kelvin.
type Temperature float64

// K returns the temperature in kelvin.
func (t Temperature) K() int {
	return int(t)
}

// C returns the temperature in degrees celcius.
func (t Temperature) C() int {
	return int(float64(t) - 273.15)
}

// F returns the temperature in degrees fahrenheit.
func (t Temperature) F() int {
	c := float64(t) - 273.15
	return int(c*1.8 + 32)
}

// Pressure provides unit conversions for pressure,
// and stores the temperature in millibar.
type Pressure float64

// Millibar returns pressure in millibars (hPa).
func (p Pressure) Millibar() float64 {
	return float64(p)
}

// Pascal returns pressure in pascals.
func (p Pressure) Pascal() float64 {
	return p.Millibar() * 100
}

// Atm returns pressure in atmospheres.
func (p Pressure) Atm() float64 {
	return p.Millibar() * 0.000986923
}

// Torr returns pressure in torr. ~= mmHg.
func (p Pressure) Torr() float64 {
	return p.Millibar() * 0.750062
}

// Psi returns pressure in pounds per square inch.
func (p Pressure) Psi() float64 {
	return p.Millibar() * 0.01450377
}

// Speed provides unit conversions for speed,
// and stores the speed in meters per second.
type Speed float64

// Ms returns the speed in meters per second.
func (s Speed) Ms() float64 {
	return float64(s)
}

// Kmh returns the speed in kilometers per hour.
func (s Speed) Kmh() float64 {
	return s.Ms() * 3.6
}

// Mph returns the speed in miles per hour.
func (s Speed) Mph() float64 {
	return s.Ms() * 2.23694
}

// Knots returns the speed in knots.
func (s Speed) Knots() float64 {
	return s.Ms() * 1.94384
}

// Direction represents a compass direction stored as degrees.
type Direction int

// Deg returns the direction in meteorological degrees.
func (d Direction) Deg() int {
	return int(d)
}

// Cardinal returns the cardinal direction.
func (d Direction) Cardinal() string {
	cardinal := ""
	deg := d.Deg()
	m := 34
	// primary cardinal direction first. N, E, S, W.
	switch {
	case deg < m || deg > 360-m:
		cardinal = "N"
	case 90-m < deg && deg < 90+m:
		cardinal = "E"
	case 180-m < deg && deg < 180+m:
		cardinal = "S"
	case 270-m < deg && deg < 270+m:
		cardinal = "W"
	}
	// Now append the midway points. NE, NW, SE, SW.
	switch {
	case 45-m < deg && deg < 45+m:
		cardinal += "NE"
	case 135-m < deg && deg < 135+m:
		cardinal += "SE"
	case 225-m < deg && deg < 225+m:
		cardinal += "SW"
	case 315-m < deg && deg < 315+m:
		cardinal += "NW"
	}
	return cardinal
}

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
	Stream() <-chan *bar.Output
	Click(e bar.Event)
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
	m.SetWorker(m.loop)
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

func (m *module) loop() error {
	for {
		weather, err := m.provider.GetWeather()
		if err != nil {
			return err
		}
		if weather != nil {
			// nil weather means unchanged.
			m.lastWeather = *weather
			m.Output(m.outputFunc(m.lastWeather))
		}
		time.Sleep(m.refreshInterval)
	}
}
