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
Package darksky provides weather using the Dark Sky API,
available at https://darksky.net/.
*/
package darksky

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/soumya92/barista/modules/weather"
)

// Config represents Dark Sky API configuration
// from which a weather.Provider can be built.
type Config struct {
	lat    float64
	lon    float64
	apiKey string
}

// New creates an empty configuration.
func New() *Config {
	return &Config{}
}

// APIKey sets the API key.
func (c *Config) APIKey(apiKey string) *Config {
	c.apiKey = apiKey
	return c
}

// Coords sets the lat/lon co-ordinates.
func (c *Config) Coords(lat, lon float64) *Config {
	c.lat = lat
	c.lon = lon
	return c
}

// Provider wraps a Dark Sky API url so that
// it can be used as a weather.Provider.
type Provider string

// Build builds a weather provider from the configuration.
func (c *Config) Build() weather.Provider {
	// Build the Dark Sky URL.
	qp := url.Values{}
	qp.Add("exclude", "minutely,hourly,alerts,flags")
	qp.Add("units", "us")
	dsURL := url.URL{
		Scheme:   "https",
		Host:     "api.darksky.net",
		Path:     fmt.Sprintf("/forecast/%s/%f,%f", c.apiKey, c.lat, c.lon),
		RawQuery: qp.Encode(),
	}
	return Provider(dsURL.String())
}

// dsWeather represents a dark sky json response.
type dsWeather struct {
	Latitude  float64
	Longitude float64
	Currently struct {
		CloudCover  float64
		Humidity    float64
		Icon        string
		Pressure    float64
		Summary     string
		Temperature float64
		Time        int64
		WindBearing int
		WindSpeed   float64
	}
	Daily struct {
		Data []struct {
			SunriseTime int64
			SunsetTime  int64
		}
	}
}

func getCondition(icon string) weather.Condition {
	switch icon {
	case "thunderstorm":
		return weather.ConditionThunderstorm
	case "rain":
		return weather.ConditionRain
	case "snow":
		return weather.ConditionSnow
	case "sleet":
		return weather.ConditionSleet
	case "fog":
		return weather.ConditionFog
	case "clear-day", "clear-night":
		return weather.ConditionClear
	case "cloudy":
		return weather.ConditionCloudy
	case "partly-cloudy-day", "partly-cloudy-night":
		return weather.ConditionOvercast
	case "tornado":
		return weather.ConditionTornado
	case "wind":
		return weather.ConditionWindy
	case "hail":
		return weather.ConditionHail
	}
	return weather.ConditionUnknown
}

// GetWeather gets weather information from OpenWeatherMap.
func (ds Provider) GetWeather() (*weather.Weather, error) {
	response, err := http.Get(string(ds))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	d := dsWeather{}
	err = json.NewDecoder(response.Body).Decode(&d)
	if err != nil {
		return nil, err
	}
	w := weather.Weather{
		City:        fmt.Sprintf("%f,%f", d.Latitude, d.Longitude),
		Condition:   getCondition(d.Currently.Icon),
		Description: d.Currently.Summary,
		Temperature: weather.TemperatureFromF(d.Currently.Temperature),
		Humidity:    d.Currently.Humidity,
		Pressure:    weather.PressureFromMillibar(d.Currently.Pressure),
		CloudCover:  d.Currently.CloudCover,
		Updated:     time.Unix(d.Currently.Time, 0),
		Wind: weather.Wind{
			weather.SpeedFromMph(d.Currently.WindSpeed),
			weather.Direction(d.Currently.WindBearing),
		},
		Attribution: "Dark Sky",
	}
	if len(d.Daily.Data) >= 1 {
		w.Sunrise = time.Unix(d.Daily.Data[0].SunriseTime, 0)
		w.Sunset = time.Unix(d.Daily.Data[0].SunsetTime, 0)
	}
	return &w, nil
}
