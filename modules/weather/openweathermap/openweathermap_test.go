// Copyright 2020 Google Inc.
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

package openweathermap

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/soumya92/barista/modules/weather"
	"github.com/soumya92/barista/testing/cron"
	testServer "github.com/soumya92/barista/testing/httpserver"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"
)

var ts *httptest.Server

func TestMain(m *testing.M) {
	ts = testServer.New()
	defer ts.Close()
	os.Exit(m.Run())
}

func TestGood(t *testing.T) {
	wthr, err := Provider(ts.URL + "/tpl/good.json?id=803&cond=Cloudy&desc=broken+clouds").GetWeather()
	require.NoError(t, err)
	require.NotNil(t, wthr)
	require.Equal(t, weather.Weather{
		Location:    "Cairns",
		Condition:   weather.Cloudy,
		Description: "broken clouds",
		Humidity:    0.83,
		Pressure:    1019 * unit.Millibar,
		Temperature: unit.FromKelvin(293.25),
		Wind: weather.Wind{
			Speed:     5.1 * unit.MetersPerSecond,
			Direction: weather.Direction(150),
		},
		CloudCover:  0.75,
		Sunrise:     time.Unix(1435610796, 0),
		Sunset:      time.Unix(1435650870, 0),
		Updated:     time.Unix(1435658272, 0),
		Attribution: "OpenWeatherMap",
	}, wthr)
}

func TestErrors(t *testing.T) {
	_, err := Provider(ts.URL + "/static/bad.json").GetWeather()
	require.Error(t, err, "bad json")

	_, err = Provider(ts.URL + "/code/401").GetWeather()
	require.Error(t, err, "http error")

	_, err = Provider(ts.URL + "/static/empty.json").GetWeather()
	require.Error(t, err, "valid json but bad response")

	_, err = Provider(ts.URL + "/redir").GetWeather()
	require.Error(t, err, "http error")
}

func TestConditions(t *testing.T) {
	for _, tc := range []struct {
		owmCondition string
		description  string
		expected     weather.Condition
	}{
		{"200", "thunderstorm with light rain", weather.Thunderstorm},
		{"201", "thunderstorm with rain", weather.Thunderstorm},
		{"202", "thunderstorm with heavy rain", weather.Thunderstorm},
		{"210", "light thunderstorm", weather.Thunderstorm},
		{"211", "thunderstorm", weather.Thunderstorm},
		{"212", "heavy thunderstorm", weather.Thunderstorm},
		{"221", "ragged thunderstorm", weather.Thunderstorm},
		{"230", "thunderstorm with light drizzle", weather.Thunderstorm},
		{"231", "thunderstorm with drizzle", weather.Thunderstorm},
		{"232", "thunderstorm with heavy drizzle", weather.Thunderstorm},
		{"300", "light intensity drizzle", weather.Drizzle},
		{"301", "drizzle", weather.Drizzle},
		{"302", "heavy intensity drizzle", weather.Drizzle},
		{"310", "light intensity drizzle rain", weather.Drizzle},
		{"311", "drizzle rain", weather.Drizzle},
		{"312", "heavy intensity drizzle rain", weather.Drizzle},
		{"313", "shower rain and drizzle", weather.Drizzle},
		{"314", "heavy shower rain and drizzle", weather.Drizzle},
		{"321", "shower drizzle", weather.Drizzle},
		{"500", "light rain", weather.Rain},
		{"501", "moderate rain", weather.Rain},
		{"502", "heavy intensity rain", weather.Rain},
		{"503", "very heavy rain", weather.Rain},
		{"504", "extreme rain", weather.Rain},
		{"511", "freezing rain", weather.Rain},
		{"520", "light intensity shower rain", weather.Rain},
		{"521", "shower rain", weather.Rain},
		{"522", "heavy intensity shower rain", weather.Rain},
		{"531", "ragged shower rain", weather.Rain},
		{"600", "light snow", weather.Snow},
		{"601", "snow", weather.Snow},
		{"602", "heavy snow", weather.Snow},
		{"611", "sleet", weather.Sleet},
		{"612", "shower sleet", weather.Sleet},
		{"615", "light rain and snow", weather.Snow},
		{"616", "rain and snow", weather.Snow},
		{"620", "light shower snow", weather.Snow},
		{"621", "shower snow", weather.Snow},
		{"622", "heavy shower snow", weather.Snow},
		{"701", "mist", weather.Mist},
		{"711", "smoke", weather.Smoke},
		{"721", "haze", weather.Haze},
		{"731", "sand, dust whirls", weather.Whirls},
		{"741", "fog", weather.Fog},
		{"751", "sand", weather.Smoke},
		{"761", "dust", weather.Smoke},
		{"762", "volcanic ash", weather.Smoke},
		{"771", "squalls", weather.Windy},
		{"781", "tornado", weather.Tornado},
		{"800", "clear sky", weather.Clear},
		{"801", "few clouds", weather.PartlyCloudy},
		{"802", "scattered clouds", weather.PartlyCloudy},
		{"803", "broken clouds", weather.Cloudy},
		{"804", "overcast clouds", weather.Overcast},
		// Not documented at OWM.
		{"900", "tornado", weather.Tornado},
		{"901", "tropical storm", weather.TropicalStorm},
		{"902", "hurricane", weather.Hurricane},
		{"903", "cold", weather.Cold},
		{"904", "hot", weather.Hot},
		{"905", "windy", weather.Windy},
		{"906", "hail", weather.Hail},
		// Unknown condition.
		{"0", "unknown", weather.ConditionUnknown},
	} {
		wthr, _ := Provider(ts.URL + "/tpl/good.json?id=" + tc.owmCondition).GetWeather()
		require.Equal(t, tc.expected, wthr.Condition,
			"OWM %s (%s)", tc.description, tc.owmCondition)
	}
}

func TestProviderBuilder(t *testing.T) {
	for _, tc := range []struct {
		expected    string
		actual      weather.Provider
		description string
	}{
		{"appid=foo&id=1234", New("foo").CityID("1234"), "CityID"},
		{"appid=foo&q=London%2CUK", New("foo").CityName("London", "UK"), "CityName"},
		{"appid=foo&lat=10.000000&lon=40.000000", New("foo").Coords(10.0, 40.0), "Coords"},
		{"appid=foo&zip=85719%2CUS", New("foo").Zipcode("85719", "US"), "Zipcode"},
	} {
		expected := "https://api.openweathermap.org/data/2.5/weather?" + tc.expected
		require.Equal(t, expected, string(tc.actual.(Provider)), tc.description)
	}
}

func TestLive(t *testing.T) {
	cron.Test(t, func() error {
		wthr, err := New(os.Getenv("WEATHER_OWM_API_KEY")).
			Zipcode("94043", "US").
			GetWeather()
		if err != nil {
			return err
		}
		require.NotNil(t, wthr)
		return nil
	})
}
