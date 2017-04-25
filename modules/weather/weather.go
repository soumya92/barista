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
// It uses OpenWeatherMap's API with a user-specified key.
package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

// Temperature provides unit conversions for temperature.
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

// Pressure provides unit conversions for pressure.
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

// Speed provides unit conversions for speed.
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

// APIKey sets the api key for openweathermap.
// Get yours from https://home.openweathermap.org/users/sign_up
type APIKey string

func (a APIKey) apply(m *module) {
	m.apiKey = string(a)
}

// RefreshInterval configures the polling frequency for OWM.
// Since free users can only make 60 requests per hour,
// and the weather doesn't change that frequently,
// set this to at least 5 minutes.
type RefreshInterval time.Duration

func (r RefreshInterval) apply(m *module) {
	m.refreshInterval = time.Duration(r)
}

// Location encodes different types of location queries for OWM.
type Location interface {
	addTo(*url.Values)
}

// CityID queries OWM by city id. Recommended.
type CityID string

func (c CityID) addTo(qp *url.Values) {
	qp.Add("id", string(c))
}

// CityName queries OWM using a named city. Least accurate.
type CityName struct {
	City, Country string
}

func (c CityName) addTo(qp *url.Values) {
	qp.Add("q", fmt.Sprintf("%s,%s", c.City, c.Country))
}

// Coords queries OWM using lat/lon co-ordinates.
type Coords struct {
	Lat, Lon float64
}

func (c Coords) addTo(qp *url.Values) {
	qp.Add("lat", fmt.Sprintf("%f", c.Lat))
	qp.Add("lon", fmt.Sprintf("%f", c.Lon))
}

// Zipcode queries OWM using a zip code or post code and country.
type Zipcode struct {
	Zip, Country string
}

func (z Zipcode) addTo(qp *url.Values) {
	qp.Add("zip", fmt.Sprintf("%s,%s", z.Zip, z.Country))
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
	location        Location
	apiKey          string
	refreshInterval time.Duration
	outputFunc      func(Weather) *bar.Output
	// cache last weather info for click handler.
	lastWeather Weather
}

// New constructs an instance of the weather module with the provided configuration.
func New(location Location, config ...Config) Module {
	m := &module{
		Base: base.New(),
		// Location is required, so it's a param instead of being an optional Config.
		location: location,
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

// owmWeather represents an openweathermap json response.
type owmWeather struct {
	Weather []struct {
		ID          int
		Main        string
		Description string
		Icon        string
	}
	Main struct {
		Temp     float64
		Pressure float64
		Humidity float64
		TempMin  float64
		TempMax  float64
	}
	Wind struct {
		Speed float64
		Deg   float64
		Gust  float64
	}
	Clouds struct {
		All float64
	}
	Sys struct {
		Sunrise int64
		Sunset  int64
	}
	Name string
	Dt   int64
}

// A compromise between OWN's not detailed enough conditions
// and their overly detailed descriptions.
var owmConditions = map[int]Condition{
	200: ConditionThunderstorm,
	201: ConditionThunderstorm,
	202: ConditionThunderstorm,
	210: ConditionThunderstorm,
	211: ConditionThunderstorm,
	212: ConditionThunderstorm,
	221: ConditionThunderstorm,
	230: ConditionThunderstorm,
	231: ConditionThunderstorm,
	232: ConditionThunderstorm,
	300: ConditionDrizzle,
	301: ConditionDrizzle,
	302: ConditionDrizzle,
	310: ConditionDrizzle,
	311: ConditionDrizzle,
	312: ConditionDrizzle,
	321: ConditionDrizzle,
	500: ConditionRain,
	501: ConditionRain,
	502: ConditionRain,
	503: ConditionRain,
	504: ConditionRain,
	511: ConditionRain,
	520: ConditionRain,
	521: ConditionRain,
	522: ConditionRain,
	600: ConditionSnow,
	601: ConditionSnow,
	602: ConditionSnow,
	611: ConditionSleet,
	621: ConditionSnow,
	701: ConditionMist,
	711: ConditionSmoke,
	721: ConditionHaze,
	731: ConditionWhirls,
	741: ConditionFog,
	800: ConditionClear,
	801: ConditionCloudy,
	802: ConditionCloudy,
	803: ConditionCloudy,
	804: ConditionOvercast,
	900: ConditionTornado,
	901: ConditionTropicalStorm,
	902: ConditionHurricane,
	903: ConditionCold,
	904: ConditionHot,
	905: ConditionWindy,
	906: ConditionHail,
}

func (m *module) loop() error {
	// Build the OWM URL.
	qp := url.Values{}
	qp.Add("appid", m.apiKey)
	m.location.addTo(&qp)
	owmURL := url.URL{
		Scheme:   "http",
		Host:     "api.openweathermap.org",
		Path:     "/data/2.5/weather",
		RawQuery: qp.Encode(),
	}
	url := owmURL.String()
	for {
		if err := m.updateWeather(url); err != nil {
			return err
		}
		m.Output(m.outputFunc(m.lastWeather))
		time.Sleep(m.refreshInterval)
	}
}

func (m *module) updateWeather(url string) error {
	response, err := http.Get(url)
	if err != nil {
		// Treat http errors as unchanged weather.
		return nil
	}
	defer response.Body.Close()
	o := owmWeather{}
	err = json.NewDecoder(response.Body).Decode(&o)
	if err != nil {
		return err
	}
	if len(o.Weather) < 1 {
		return fmt.Errorf("Bad response from OWM")
	}
	condition, ok := owmConditions[o.Weather[0].ID]
	if !ok {
		condition = ConditionUnknown
	}
	m.lastWeather = Weather{
		City:        o.Name,
		Condition:   condition,
		Description: o.Weather[0].Description,
		Temperature: Temperature(o.Main.Temp),
		Humidity:    o.Main.Humidity,
		Pressure:    Pressure(o.Main.Pressure),
		Wind:        Wind{Speed(o.Wind.Speed), Direction(int(o.Wind.Deg))},
		CloudCover:  o.Clouds.All,
		Sunrise:     time.Unix(o.Sys.Sunrise, 0),
		Sunset:      time.Unix(o.Sys.Sunset, 0),
		Updated:     time.Unix(o.Dt, 0),
	}
	return nil
}
