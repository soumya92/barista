package apixu

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"barista.run/modules/weather"
	"barista.run/testing/cron"
	testServer "barista.run/testing/httpserver"

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
	wthr, err := Provider(ts.URL + "/static/good.json").GetWeather()
	require.NoError(t, err)
	require.NotNil(t, wthr)
	require.Equal(t, weather.Weather{
		Location:    "Greenville, South Carolina, USA",
		Condition:   weather.Rain,
		Description: "Light rain",
		Humidity:    0.96,
		Pressure:    1016.0 * unit.Millibar,
		Temperature: unit.FromFahrenheit(48.0),
		Wind: weather.Wind{
			Speed:     unit.Speed(19.1) * unit.MetersPerSecond,
			Direction: weather.Direction(40),
		},
		CloudCover:  1.0,
		Updated:     time.Unix(1544845514, 0),
		Attribution: "Apixu",
	}, wthr)
}

func TestErrors(t *testing.T) {
	_, err := Provider(ts.URL + "/code/400").GetWeather()
	require.Error(t, err, "bad request")

	_, err = Provider(ts.URL + "/code/401").GetWeather()
	require.Error(t, err, "authentication error")

	_, err = Provider(ts.URL + "/code/403").GetWeather()
	require.Error(t, err, "API key exceeded monthly quota")
}

func TestConditions(t *testing.T) {
	for _, tc := range []struct {
		apixuCondition string
		description    string
		expected       weather.Condition
	}{
		{"1000", "Sunny", weather.Clear},
		{"1003", "Partly cloudy", weather.PartlyCloudy},
		{"1006", "Cloudy", weather.Cloudy},
		{"1009", "Overcast", weather.Overcast},
		{"1030", "Mist", weather.Mist},
		{"1063", "Patchy rain possible", weather.Rain},
		{"1066", "Patchy snow possible", weather.Snow},
		{"1069", "Patchy sleet possible", weather.Sleet},
		{"1072", "Patchy freezing drizzle possible", weather.Drizzle},
		{"1087", "Thundery outbreaks possible", weather.Thunderstorm},
		{"1114", "Blowing snow", weather.Snow},
		{"1117", "Blizzard", weather.Snow},
		{"1135", "Fog", weather.Fog},
		{"1147", "Freezing fog", weather.Fog},
		{"1150", "Patchy light drizzle", weather.Drizzle},
		{"1153", "Light drizzle", weather.Drizzle},
		{"1168", "Freezing drizzle", weather.Drizzle},
		{"1171", "Heavy freezing drizzle", weather.Drizzle},
		{"1180", "Patchy light rain", weather.Rain},
		{"1183", "Light rain", weather.Rain},
		{"1186", "Moderate rain at times", weather.Rain},
		{"1189", "Moderate rain", weather.Rain},
		{"1192", "Heavy rain at times", weather.Rain},
		{"1195", "Heavy rain", weather.Rain},
		{"1198", "Light freezing rain", weather.Rain},
		{"1201", "Moderate or heavy freezing rain", weather.Rain},
		{"1204", "Light sleet", weather.Sleet},
		{"1207", "Moderate or heavy sleet", weather.Sleet},
		{"1210", "Patchy light snow", weather.Snow},
		{"1213", "Light snow", weather.Snow},
		{"1216", "Patchy moderate snow", weather.Snow},
		{"1219", "Moderate snow", weather.Snow},
		{"1222", "Patchy heavy snow", weather.Snow},
		{"1225", "Heavy snow", weather.Snow},
		{"1237", "Ice pellets", weather.Hail},
		{"1240", "Light rain shower", weather.Rain},
		{"1243", "Moderate or heavy rain shower", weather.Rain},
		{"1246", "Torrential rain shower", weather.Rain},
		{"1249", "Light sleet showers", weather.Sleet},
		{"1252", "Moderate or heavy sleet showers", weather.Sleet},
		{"1255", "Light snow showers", weather.Snow},
		{"1258", "Moderate or heavy snow showers", weather.Snow},
		{"1261", "Light showers of ice pellets", weather.Hail},
		{"1264", "Moderate or heavy showers of ice pellets", weather.Hail},
		{"1273", "Patchy light rain with thunder", weather.Thunderstorm},
		{"1276", "Moderate or heavy rain with thunder", weather.Thunderstorm},
		{"1279", "Patchy light snow with thunder", weather.Snow},
		{"1282", "Moderate or heavy snow with thunder", weather.Snow},
		// Unknown condition.
		{"0", "unknown", weather.ConditionUnknown},
	} {
		wthr, _ := Provider(ts.URL + "/tpl/conditions.json?code=" + tc.apixuCondition).GetWeather()
		require.Equal(t, tc.expected, wthr.Condition,
			"Apixu %s (%s)", tc.description, tc.apixuCondition)
	}
}

func TestLive(t *testing.T) {
	cron.Test(t, func() error {
		wthr, err := New(os.Getenv("APIXU_API_KEY")).
			Query("29617").
			GetWeather()
		if err != nil {
			return err
		}
		require.NotNil(t, wthr)
		return nil
	})
}
