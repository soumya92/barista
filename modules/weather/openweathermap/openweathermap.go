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
Package openweathermap provides weather using the OpenWeatherMap API,
available at https://openweathermap.org/api.
*/
package openweathermap // import "barista.run/modules/weather/openweathermap"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"barista.run/modules/weather"

	"github.com/martinlindhe/unit"
)

// Config represents open weather map API configuration
// from which a weather.Provider can be built.
type Config struct {
	query  [][2]string
	apiKey string
}

// CityID queries OWM by city id. Recommended.
func CityID(cityID string) *Config {
	return &Config{query: [][2]string{
		{"id", cityID},
	}}
}

// CityName queries OWM using a named city. Least accurate.
func CityName(city, country string) *Config {
	return &Config{query: [][2]string{
		{"q", fmt.Sprintf("%s,%s", city, country)},
	}}
}

// Coords queries OWM using lat/lon co-ordinates.
func Coords(lat, lon float64) *Config {
	return &Config{query: [][2]string{
		{"lat", fmt.Sprintf("%.6f", lat)},
		{"lon", fmt.Sprintf("%.6f", lon)},
	}}
}

// Zipcode queries OWM using a zip code or post code and country.
func Zipcode(zip, country string) *Config {
	return &Config{query: [][2]string{
		{"zip", fmt.Sprintf("%s,%s", zip, country)},
	}}
}

// APIKey sets the API key if a different api key is preferred.
func (c *Config) APIKey(apiKey string) *Config {
	c.apiKey = apiKey
	return c
}

// Provider wraps an open weather map API url so that
// it can be used as a weather.Provider.
type Provider string

// Build builds a weather provider from the configuration.
func (c *Config) Build() weather.Provider {
	// Build the OWM URL.
	qp := url.Values{}
	apiKey := c.apiKey
	// Use barista's API key if no API key was explicitly provided.
	if apiKey == "" {
		apiKey = "9c51204f81fc8e1998981de83a7cabc9"
	}
	qp.Add("appid", apiKey)
	for _, value := range c.query {
		qp.Add(value[0], value[1])
	}
	owmURL := url.URL{
		Scheme:   "http",
		Host:     "api.openweathermap.org",
		Path:     "/data/2.5/weather",
		RawQuery: qp.Encode(),
	}
	return Provider(owmURL.String())
}

// owmWeather represents an openweathermap json response.
type owmWeather struct {
	Weather []struct {
		ID          int
		Description string
	}
	Main struct {
		Temp     float64
		Pressure float64
		Humidity float64
	}
	Wind struct {
		Speed float64
		Deg   float64
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

func getCondition(owmCondition int) weather.Condition {
	switch owmCondition {
	case 611, 612:
		return weather.Sleet
	case 701:
		return weather.Mist
	case 711, 751, 761, 762:
		return weather.Smoke
	case 721:
		return weather.Haze
	case 731:
		return weather.Whirls
	case 741:
		return weather.Fog
	case 800:
		return weather.Clear
	case 801, 802:
		return weather.PartlyCloudy
	case 803:
		return weather.Cloudy
	case 804:
		return weather.Overcast
	case 900, 781:
		return weather.Tornado
	case 901:
		return weather.TropicalStorm
	case 902:
		return weather.Hurricane
	case 903:
		return weather.Cold
	case 904:
		return weather.Hot
	case 905, 771:
		return weather.Windy
	case 906:
		return weather.Hail
	}
	if owmCondition >= 200 && owmCondition < 300 {
		return weather.Thunderstorm
	} else if owmCondition >= 300 && owmCondition < 500 {
		return weather.Drizzle
	} else if owmCondition >= 500 && owmCondition < 600 {
		return weather.Rain
	} else if owmCondition >= 600 && owmCondition < 700 {
		return weather.Snow
	}
	return weather.ConditionUnknown
}

// GetWeather gets weather information from OpenWeatherMap.
func (owm Provider) GetWeather() (weather.Weather, error) {
	response, err := http.Get(string(owm))
	if err != nil {
		return weather.Weather{}, err
	}
	defer response.Body.Close()
	o := owmWeather{}
	err = json.NewDecoder(response.Body).Decode(&o)
	if err != nil {
		return weather.Weather{}, err
	}
	if len(o.Weather) < 1 {
		return weather.Weather{}, fmt.Errorf("Bad response from OWM")
	}
	return weather.Weather{
		Location:    o.Name,
		Condition:   getCondition(o.Weather[0].ID),
		Description: o.Weather[0].Description,
		Temperature: unit.FromKelvin(o.Main.Temp),
		Humidity:    float64(o.Main.Humidity) / 100.0,
		Pressure:    unit.Pressure(o.Main.Pressure) * unit.Millibar,
		CloudCover:  float64(o.Clouds.All) / 100.0,
		Sunrise:     time.Unix(o.Sys.Sunrise, 0),
		Sunset:      time.Unix(o.Sys.Sunset, 0),
		Updated:     time.Unix(o.Dt, 0),
		Wind: weather.Wind{
			Speed:     unit.Speed(o.Wind.Speed) * unit.MetersPerSecond,
			Direction: weather.Direction(int(o.Wind.Deg)),
		},
		Attribution: "OpenWeatherMap",
	}, nil
}
