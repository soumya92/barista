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
package openweathermap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/soumya92/barista/modules/weather"
)

// Config represents open weather map API configuration
// from which a weather.Provider can be built.
type Config struct {
	query  map[string]string
	apiKey string
}

// New creates a default configuration with barista's API key.
func New() *Config {
	return &Config{
		// Barista API Key, can be overridden by users if they want to use their own.
		apiKey: "9c51204f81fc8e1998981de83a7cabc9",
	}
}

// APIKey sets the API key if a different api key is preferred.
func (c *Config) APIKey(apiKey string) *Config {
	c.apiKey = apiKey
	return c
}

// CityID queries OWM by city id. Recommended.
func (c *Config) CityID(cityID string) *Config {
	c.query = map[string]string{"id": cityID}
	return c
}

// CityName queries OWM using a named city. Least accurate.
func (c *Config) CityName(city, country string) *Config {
	c.query = map[string]string{
		"q": fmt.Sprintf("%s,%s", city, country),
	}
	return c
}

// Coords queries OWM using lat/lon co-ordinates.
func (c *Config) Coords(lat, lon float64) *Config {
	c.query = map[string]string{
		"lat": fmt.Sprintf("%f", lat),
		"lon": fmt.Sprintf("%f", lon),
	}
	return c
}

// Zipcode queries OWM using a zip code or post code and country.
func (c *Config) Zipcode(zip, country string) *Config {
	c.query = map[string]string{
		"zip": fmt.Sprintf("%s,%s", zip, country),
	}
	return c
}

// Provider wraps an open weather map API url so that
// it can be used as a weather.Provider.
type Provider string

// Build builds a weather provider from the configuration.
func (c *Config) Build() weather.Provider {
	// Build the OWM URL.
	qp := url.Values{}
	qp.Add("appid", c.apiKey)
	for key, value := range c.query {
		qp.Add(key, value)
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
	case 611:
		return weather.ConditionSleet
	case 701:
		return weather.ConditionMist
	case 711:
		return weather.ConditionSmoke
	case 721:
		return weather.ConditionHaze
	case 731:
		return weather.ConditionWhirls
	case 741:
		return weather.ConditionFog
	case 800:
		return weather.ConditionClear
	case 801, 802, 803:
		return weather.ConditionCloudy
	case 804:
		return weather.ConditionOvercast
	case 900:
		return weather.ConditionTornado
	case 901:
		return weather.ConditionTropicalStorm
	case 902:
		return weather.ConditionHurricane
	case 903:
		return weather.ConditionCold
	case 904:
		return weather.ConditionHot
	case 905:
		return weather.ConditionWindy
	case 906:
		return weather.ConditionHail
	}
	if owmCondition >= 200 && owmCondition < 300 {
		return weather.ConditionThunderstorm
	} else if owmCondition >= 300 && owmCondition < 500 {
		return weather.ConditionDrizzle
	} else if owmCondition >= 500 && owmCondition < 600 {
		return weather.ConditionRain
	} else if owmCondition >= 600 && owmCondition < 700 {
		return weather.ConditionSnow
	}
	return weather.ConditionUnknown
}

// GetWeather gets weather information from OpenWeatherMap.
func (owm Provider) GetWeather() (*weather.Weather, error) {
	response, err := http.Get(string(owm))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	o := owmWeather{}
	err = json.NewDecoder(response.Body).Decode(&o)
	if err != nil {
		return nil, err
	}
	if len(o.Weather) < 1 {
		return nil, fmt.Errorf("Bad response from OWM")
	}
	return &weather.Weather{
		City:        o.Name,
		Condition:   getCondition(o.Weather[0].ID),
		Description: o.Weather[0].Description,
		Temperature: weather.TemperatureFromK(o.Main.Temp),
		Humidity:    o.Main.Humidity,
		Pressure:    weather.PressureFromMillibar(o.Main.Pressure),
		CloudCover:  o.Clouds.All,
		Sunrise:     time.Unix(o.Sys.Sunrise, 0),
		Sunset:      time.Unix(o.Sys.Sunset, 0),
		Updated:     time.Unix(o.Dt, 0),
		Wind: weather.Wind{
			weather.SpeedFromMs(o.Wind.Speed),
			weather.Direction(int(o.Wind.Deg)),
		},
		Attribution: "OpenWeatherMap",
	}, nil
}
