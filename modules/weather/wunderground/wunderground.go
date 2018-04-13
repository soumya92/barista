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

/*
Package wunderground provides weather using the Weather Underground API,
available at https://www.wunderground.com/weather/api/.
*/
package wunderground

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/modules/weather"
)

// Config represents Weather Underground API configuration
// from which a weather.Provider can be built.
type Config struct {
	query  string
	apiKey string
}

// USCity queries by a US City and State.
func USCity(city, state string) *Config {
	return &Config{query: fmt.Sprintf("%s/%s", state, city)}
}

// USZipCode queries by a US Zip Code.
func USZipCode(zipcode string) *Config {
	return &Config{query: zipcode}
}

// City queries by a city and country
func City(city, country string) *Config {
	return &Config{query: fmt.Sprintf("%s/%s", country, city)}
}

// Coords queries by lat/lon co-ordinates.
func Coords(lat, lon float64) *Config {
	return &Config{query: fmt.Sprintf("%f,%f", lat, lon)}
}

// Airport queries by airport code (e.g. KSEA).
func Airport(code string) *Config {
	return &Config{query: code}
}

// PWS queries by personal weather station id (e.g. KCASANFR70).
func PWS(id string) *Config {
	return &Config{query: fmt.Sprintf("pwd:%s", id)}
}

// ZMW queries by the disambiguation link if multiple locations
// match a city name. The zmw number needs to be looked up manually
// by making a request and observing the 'results' array.
func ZMW(zmw string) *Config {
	return &Config{query: fmt.Sprintf("zmw:%s", zmw)}
}

// APIKey sets the API key.
func (c *Config) APIKey(apiKey string) *Config {
	c.apiKey = apiKey
	return c
}

// Provider wraps a Weather Underground API url so that
// it can be used as a weather.Provider.
type Provider string

// Build builds a weather provider from the configuration.
func (c *Config) Build() weather.Provider {
	// Build the OWM URL.
	wURL := url.URL{
		Scheme: "http",
		Host:   "api.wunderground.com",
		Path:   fmt.Sprintf("/api/%s/conditions/q/%s.json", c.apiKey, c.query),
	}
	return Provider(wURL.String())
}

// wuWeather represents a Weather Underground json response.
type wuWeather struct {
	CurrentObservation struct {
		DisplayLocation struct {
			City string
		} `json:"display_location"`
		Weather          string
		TempC            float64 `json:"temp_c"`
		Humidity         string  `json:"relative_humidity"`
		PressureMb       string  `json:"pressure_mb"`
		ObservationEpoch string  `json:"observation_epoch"`
		WindDegrees      int     `json:"wind_degrees"`
		WindKph          float64 `json:"wind_kph"`
		Icon             string
	} `json:"current_observation"`
}

func getCondition(icon string) weather.Condition {
	switch icon {
	case "chancetstorms", "tstorms":
		return weather.Thunderstorm
	case "chancerain", "rain":
		return weather.Rain
	case "chanceflurries", "chancesnow", "flurries", "snow":
		return weather.Snow
	case "chancesleet", "sleet":
		return weather.Sleet
	case "hazy":
		return weather.Haze
	case "fog":
		return weather.Fog
	case "clear", "mostlysunny", "sunny":
		return weather.Clear
	case "cloudy", "mostlycloudy":
		return weather.Cloudy
	case "partlycloudy", "partlysunny":
		return weather.PartlyCloudy
	}
	return weather.ConditionUnknown
}

func parsePercent(percent string) float64 {
	floatVal, _ := strconv.ParseFloat(strings.TrimSuffix(percent, "%"), 64)
	return floatVal
}

func parsePressure(pressure string) unit.Pressure {
	floatVal, _ := strconv.ParseFloat(pressure, 64)
	return unit.Pressure(floatVal) * unit.Millibar
}

func parseUnixTime(unix string) time.Time {
	intVal, err := strconv.ParseInt(unix, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(intVal, 0)
}

// GetWeather gets weather information from Weather Underground.
func (wu Provider) GetWeather() (*weather.Weather, error) {
	response, err := http.Get(string(wu))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	w := wuWeather{}
	err = json.NewDecoder(response.Body).Decode(&w)
	if err != nil {
		return nil, err
	}
	return &weather.Weather{
		Location:    w.CurrentObservation.DisplayLocation.City,
		Condition:   getCondition(w.CurrentObservation.Icon),
		Description: w.CurrentObservation.Weather,
		Temperature: unit.FromCelsius(w.CurrentObservation.TempC),
		Humidity:    parsePercent(w.CurrentObservation.Humidity),
		Pressure:    parsePressure(w.CurrentObservation.PressureMb),
		Updated:     parseUnixTime(w.CurrentObservation.ObservationEpoch),
		Wind: weather.Wind{
			Speed:     unit.Speed(w.CurrentObservation.WindKph) * unit.KilometersPerHour,
			Direction: weather.Direction(w.CurrentObservation.WindDegrees),
		},
		Attribution: "Weather Underground",
	}, nil
}
