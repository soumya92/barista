// Copyright 2018 Google Inc.
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

package darksky

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/modules/weather"
	"github.com/soumya92/barista/testing/cron"
	testServer "github.com/soumya92/barista/testing/httpserver"
)

var ts *httptest.Server

func TestMain(m *testing.M) {
	ts = testServer.New()
	defer ts.Close()
	os.Exit(m.Run())
}

func TestGood(t *testing.T) {
	wthr, err := Provider(ts.URL + "/tpl/good.json?icon=rain").GetWeather()
	require.NoError(t, err)
	require.NotNil(t, wthr)
	require.Equal(t, weather.Weather{
		Location:    "42.360100,-71.058900",
		Condition:   weather.Rain,
		Description: "Drizzle",
		Humidity:    0.83,
		Pressure:    1010.34 * unit.Millibar,
		Temperature: unit.FromFahrenheit(66.1),
		Wind: weather.Wind{
			Speed:     5.59 * unit.MilesPerHour,
			Direction: weather.Direction(246),
		},
		CloudCover:  0.7,
		Sunrise:     time.Unix(1509967519, 0),
		Sunset:      time.Unix(1510003982, 0),
		Updated:     time.Unix(1509993277, 0),
		Attribution: "Dark Sky",
	}, *wthr)
}

func TestErrors(t *testing.T) {
	wthr, err := Provider(ts.URL + "/static/bad.json").GetWeather()
	require.Error(t, err, "bad json")
	require.Nil(t, wthr)

	wthr, err = Provider(ts.URL + "/code/401").GetWeather()
	require.Error(t, err, "http error")
	require.Nil(t, wthr)

	wthr, err = Provider(ts.URL + "/static/empty.json").GetWeather()
	require.Error(t, err, "valid json but bad response")
	require.Nil(t, wthr)

	wthr, err = Provider(ts.URL + "/redir").GetWeather()
	require.Error(t, err, "http error")
	require.Nil(t, wthr)
}

func TestConditions(t *testing.T) {
	for _, tc := range []struct {
		dsIcon   string
		expected weather.Condition
	}{
		{"clear-day", weather.Clear},
		{"clear-night", weather.Clear},
		{"rain", weather.Rain},
		{"snow", weather.Snow},
		{"sleet", weather.Sleet},
		{"wind", weather.Windy},
		{"fog", weather.Fog},
		{"cloudy", weather.Cloudy},
		{"partly-cloudy-day", weather.PartlyCloudy},
		{"partly-cloudy-night", weather.PartlyCloudy},
		{"hail", weather.Hail},
		{"thunderstorm", weather.Thunderstorm},
		{"tornado", weather.Tornado},
		{"other", weather.ConditionUnknown},
	} {
		wthr, _ := Provider(ts.URL + "/tpl/good.json?icon=" + tc.dsIcon).GetWeather()
		require.Equal(t, tc.expected, wthr.Condition, "DarkSky %s", tc.dsIcon)
	}
}

func TestProviderBuilder(t *testing.T) {
	for _, tc := range []struct {
		expected string
		actual   *Config
	}{
		{"/apikey/40.689200,-74.044500", Coords(40.6892, -74.0445).APIKey("apikey")},
		{"//-37.422000,122.084100", Coords(-37.4220, 122.0841)},
	} {
		expected := "https://api.darksky.net/forecast" + tc.expected +
			"?exclude=minutely%2Chourly%2Calerts%2Cflags&units=us"
		require.Equal(t, expected, string(tc.actual.Build().(Provider)))
	}
}

func TestLive(t *testing.T) {
	cron.Test(t, func(t *testing.T) {
		wthr, err := Coords(42.3601, -71.0589).
			APIKey(os.Getenv("WEATHER_DS_API_KEY")).
			Build().
			GetWeather()
		require.NoError(t, err)
		require.NotNil(t, wthr)
	})
}
