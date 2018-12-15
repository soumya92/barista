package apixu // import "barista.run/modules/weather/apixu"

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"barista.run/modules/weather"

	"github.com/martinlindhe/unit"
)

// Config represents apixu API configuration (just the API key)
// from which a weather.Provider can be built.
type Config string

// New creates a new Apixu API configuration.
func New(apiKey string) Config {
	return Config(apiKey)
}

// Query queries Apixu using zip code, lat/lon, city name, etc. (see https://www.apixu.com/doc/request.aspx)
func (c Config) Query(q string) weather.Provider {
	qp := url.Values{}
	qp.Add("key", string(c))
	qp.Add("q", q)
	apixuURL := url.URL{
		Scheme:   "http",
		Host:     "api.apixu.com",
		Path:     "/v1/current.json",
		RawQuery: qp.Encode(),
	}
	return Provider(apixuURL.String())
}

// Provider wraps an Apixu API url so that
// it can be used as a weather.Provider.
type Provider string

// apixuWeather represents an Apixu json response.
type apixuWeather struct {
	Location struct {
		Name           string  `json:"name"`
		Region         string  `json:"region"`
		Country        string  `json:"country"`
		Lat            float64 `json:"lat"`
		Lon            float64 `json:"lon"`
		TzID           string  `json:"tz_id"`
		LocaltimeEpoch int64   `json:"localtime_epoch"`
		Localtime      string  `json:"localtime"`
	} `json:"location"`

	Current struct {
		Condition struct {
			Text string `json:"text"`
			Icon string `json:"icon"`
			// Code is a code for describing weather (see https://www.apixu.com/doc/weather-conditions.aspx)
			Code int `json:"code"`
		} `json:"condition"`

		LastUpdated      string `json:"last_updated"`
		LastUpdatedEpoch int64  `json:"last_updated_epoch"`

		TempC float64 `json:"temp_c"`
		TempF float64 `json:"temp_f"`

		FeelsLikeC float64 `json:"feelslike_c"`
		FeelsLikeF float64 `json:"feelslike_f"`

		WindMPH    float64 `json:"wind_mph"`
		WindKPH    float64 `json:"wind_kph"`
		WindDegree int     `json:"wind_degree"`
		WindDir    string  `json:"wind_dir"`

		// IsDay is day (1) or night (0)
		IsDay int `json:"is_day"`

		// PressureMB is pressure in millibars
		PressureMB float64 `json:"pressure_mb"`
		// PressureIN is pressure in inches
		PressureIN float64 `json:"pressure_in"`

		// PrecipMM is precipitation in millimeters
		PrecipMM float64 `json:"precip_mm"`
		// PrecipIN is precipitation in inches
		PrecipIN float64 `json:"precip_in"`

		Humidity int `json:"humidity"`

		// Cloud is cloud cover as percentage
		Cloud int `json:"cloud"`

		// VisKM is visibility is kilometers
		VisKM float64 `json:"vis_km"`
		// VisMiles is visibility is miles
		VisMiles float64 `json:"vis_miles"`

		UV float64 `json:"uv"`
	} `json:"current"`
}

func getCondition(apixuCondition int) weather.Condition {
	switch apixuCondition {
	case 1000:
		return weather.Clear
	case 1003:
		return weather.PartlyCloudy
	case 1006:
		return weather.Cloudy
	case 1009:
		return weather.Overcast
	case 1030:
		return weather.Mist
	case 1063, 1180, 1183, 1186, 1189, 1192,
		1195, 1198, 1201, 1240, 1243, 1246:
		return weather.Rain
	case 1066, 1114, 1117, 1210, 1213, 1216, 1219,
		1222, 1225, 1255, 1258, 1279, 1282:
		return weather.Snow
	case 1069, 1204, 1207, 1249, 1252:
		return weather.Sleet
	case 1072, 1150, 1153, 1168, 1171:
		return weather.Drizzle
	case 1087, 1273, 1276:
		return weather.Thunderstorm
	case 1135, 1147:
		return weather.Fog
	case 1237, 1261, 1264:
		return weather.Hail
	}

	return weather.ConditionUnknown
}

// GetWeather gets weather information from Apixu.
func (apixuProvider Provider) GetWeather() (weather.Weather, error) {
	response, err := http.Get(string(apixuProvider))
	if err != nil {
		return weather.Weather{}, err
	}
	defer response.Body.Close()

	if response.StatusCode == 401 {
		return weather.Weather{}, fmt.Errorf("Invalid or missing API key")
	}

	if response.StatusCode == 403 {
		return weather.Weather{}, fmt.Errorf("API key exceeded monthly quota")
	}

	a := apixuWeather{}
	err = json.NewDecoder(response.Body).Decode(&a)
	if err != nil {
		return weather.Weather{}, err
	}

	return weather.Weather{
		Location: strings.Join([]string{
			a.Location.Name,
			a.Location.Region,
			a.Location.Country,
		}, ", "),
		Condition:   getCondition(a.Current.Condition.Code),
		Description: a.Current.Condition.Text,
		Temperature: unit.FromFahrenheit(a.Current.TempF),
		Humidity:    float64(a.Current.Humidity) / 100.0,
		Pressure:    unit.Pressure(a.Current.PressureMB) * unit.Millibar,
		CloudCover:  float64(a.Current.Cloud) / 100.0,
		Updated:     time.Unix(a.Current.LastUpdatedEpoch, 0),
		Wind: weather.Wind{
			Speed:     unit.Speed(a.Current.WindKPH) * unit.MetersPerSecond,
			Direction: weather.Direction(a.Current.WindDegree),
		},
		Attribution: "Apixu",
	}, nil
}
