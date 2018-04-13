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
Package metar provides weather using the METAR API from
the NOAA Aviation Digital Data Service,
available at https://www.aviationweather.gov/.
*/
package metar

import (
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/modules/weather"
)

// Config represents NOAA ADDS configuration
// from which a weather.Provider can be built.
type Config struct {
	station          string
	stripRemarks     bool
	includeFlightCat bool
}

// Station creates a configuration for the given airport code (e.g. KSEA).
func Station(station string) *Config {
	return &Config{station: station}
}

// StripRemarks strips remarks from the description.
func (c *Config) StripRemarks() *Config {
	c.stripRemarks = true
	return c
}

// IncludeFlightCat adds flight category to the description.
func (c *Config) IncludeFlightCat() *Config {
	c.includeFlightCat = true
	return c
}

// Provider wraps an ADDS XML url and configuration
// so that it can be used as a weather.Provider.
type Provider struct {
	url              string
	stripRemarks     bool
	includeFlightCat bool
}

// Build builds a weather provider from the configuration.
func (c *Config) Build() weather.Provider {
	u := url.URL{
		Scheme: "https",
		Host:   "aviationweather.gov",
		Path:   "/adds/dataserver_current/httpparam",
	}
	q := u.Query()
	q.Set("dataSource", "metars")
	q.Set("requestType", "retrieve")
	q.Set("format", "xml")
	q.Set("stationString", c.station)
	q.Set("hoursBeforeNow", "3")
	q.Set("mostRecent", "true")
	u.RawQuery = q.Encode()

	return Provider{
		url:              u.String(),
		stripRemarks:     c.stripRemarks,
		includeFlightCat: c.includeFlightCat,
	}
}

type skyCondition struct {
	SkyCover  string `xml:"sky_cover,attr"`
	CloudBase int    `xml:"cloud_base_ft_agl,attr"`
}

type metar struct {
	RawText            string         `xml:"raw_text"`
	StationID          string         `xml:"station_id"`
	ObservationTime    string         `xml:"observation_time"`
	Latitude           float64        `xml:"latitude"`
	Longitude          float64        `xml:"longitude"`
	Temperature        float64        `xml:"temp_c"`
	Dewpoint           float64        `xml:"dewpoint_c"`
	WindDirection      int            `xml:"wind_dir_degrees"`
	WindSpeed          int            `xml:"wind_speed_kt"`
	WindGust           int            `xml:"wind_gust_kt"`
	Visibility         float64        `xml:"visibility_statute_mi"`
	Altimeter          float64        `xml:"altim_in_hg"`
	SeaLevelPressure   float64        `xml:"sea_level_pressure_mb"`
	WxString           string         `xml:"wx_string"`
	SkyConditions      []skyCondition `xml:"sky_condition"`
	FlightCategory     string         `xml:"flight_category"`
	VerticalVisibility int            `xml:"vert_vis_ft"`
	StationElevation   float64        `xml:"elevation_m"`
}

func (m metar) getBarometricPressure() unit.Pressure {
	if m.SeaLevelPressure == 0.0 {
		return unit.Pressure(m.Altimeter*3386.39) * unit.Pascal
	}
	return unit.Pressure(m.SeaLevelPressure) * unit.Millibar
}

type addsResponse struct {
	Metars []metar `xml:"data>METAR"`
}

// The August-Roche-Magnus approximation to the saturation vapor pressure.
// https://en.wikipedia.org/wiki/Clausius–Clapeyron_relation
func satVaporPressure(temp float64) float64 {
	return 6.1094 * math.Exp((17.625*temp)/(temp+243.04))
}

// See also: http://andrew.rsmas.miami.edu/bmcnoldy/Humidity.html
func relativeHumidity(temp float64, dewpoint float64) float64 {
	return 100 * (satVaporPressure(dewpoint) / satVaporPressure(temp))
}

var remarksPattern = regexp.MustCompile(` RMK (.*)$`)
var tempPattern = regexp.MustCompile(`\b(\d+)/(\d+)\b`)
var preciseTempPattern = regexp.MustCompile(`\bT\d{8}\b`) // T01320072

func encodeMetarTemp(temp float64) string {
	minus := ""
	if temp < 0 {
		minus = "M"
		temp = -temp
	}

	return fmt.Sprintf("%s%04.1f", minus, temp)
}

func (m metar) encodeMetar(stripRemarks bool, includeFlightCat bool) string {
	mt := strings.TrimSpace(m.RawText)
	if stripRemarks {
		mt = remarksPattern.ReplaceAllString(mt, "")

		// Include the temperature to 0.1C, if it's included in the
		// base METAR. If it is, the METAR will have a T block, and the
		// value will already be parsed in the Temperature and Dewpoint
		// fields of the XML response.
		if preciseTempPattern.MatchString(m.RawText) {
			preciseTemp := fmt.Sprintf(
				"%s/%s",
				encodeMetarTemp(m.Temperature),
				encodeMetarTemp(m.Dewpoint))
			mt = tempPattern.ReplaceAllString(mt, preciseTemp)
		}
	}

	if includeFlightCat {
		mt = fmt.Sprintf("[%s] %s", m.FlightCategory, mt)
	}

	return mt
}

// Cloudiness, in oktas (eighths of the sky covered)
type cloudiness int

const (
	cloudsNone      cloudiness = 0
	cloudsFew                  = 2
	cloudsScattered            = 4
	cloudsBroken               = 7
	cloudsOvercast             = 8
)

var cloudinessMap = map[string]cloudiness{
	"CLR":   cloudsNone,
	"SKC":   cloudsNone,
	"CAVOK": cloudsNone,
	"FEW":   cloudsFew,
	"SCT":   cloudsScattered,
	"BKN":   cloudsBroken,
	"OVC":   cloudsOvercast,
	"OVX":   cloudsOvercast,
}

func (m metar) getCloudiness() cloudiness {
	c := cloudsNone
	for _, skyCond := range m.SkyConditions {
		coverage := cloudinessMap[skyCond.SkyCover]
		if coverage > c {
			c = coverage
		}
	}
	return c
}

func (m metar) getCloudCover() float64 {
	return float64(m.getCloudiness()) / 8.0
}

var thunderstormPattern = regexp.MustCompile(`\b[+-]?TS(..)?\b`) // +TSRA, -TSSN, TS
var precipPattern = regexp.MustCompile(`\b([+-]?)(..)?(..)\b`)

// see http://weather.cod.edu/notes/metar.html
var precipMapping = map[string]weather.Condition{
	"DZ": weather.Drizzle,
	"RA": weather.Rain,
	"SN": weather.Snow,
	"SG": weather.Snow,
	"IC": weather.Sleet,
	"PL": weather.Sleet,
	"GR": weather.Hail,
	"GS": weather.Hail,
	"UP": weather.ConditionUnknown,
	"BR": weather.Mist,
	"FG": weather.Fog,
	"FU": weather.Smoke,
	"VA": weather.Smoke,
	"DU": weather.Whirls,
	"SA": weather.Whirls,
	"HZ": weather.Haze,
	"PY": weather.Haze,
	"FC": weather.Tornado,
	"PO": weather.Whirls,
	"SQ": weather.Windy,
	"SS": weather.Whirls,
}

var precipOrder = []weather.Condition{
	weather.Tornado,
	weather.Hail,
	weather.Whirls,
	weather.Windy,
	weather.Smoke,
	weather.Sleet,
	weather.Snow,
	weather.Fog,
	weather.Haze,
	weather.Rain,
	weather.Drizzle,
	weather.ConditionUnknown,
}

func (m metar) getCondition() weather.Condition {
	// Check for thunderstorms first, since that's considered a modifier,
	// rather than a weather condition in and of itself.
	if thunderstormPattern.MatchString(m.WxString) {
		return weather.Thunderstorm
	}

	// Find all of the weather conditions present in the METAR.
	hasCondition := map[weather.Condition]bool{}
	for _, wx := range precipPattern.FindAllStringSubmatch(m.WxString, 0) {
		precip := wx[3]
		hasCondition[precipMapping[precip]] = true
	}

	for _, condition := range precipOrder {
		if hasCondition[condition] {
			return condition
		}
	}

	// No precipitation, look at clouds.
	switch m.getCloudiness() {
	case cloudsOvercast:
		return weather.Overcast
	case cloudsBroken:
		return weather.Cloudy
	case cloudsScattered:
	case cloudsFew:
		return weather.PartlyCloudy
	case cloudsNone:
		return weather.Clear
	}

	return weather.Clear
}

// GetWeather gets weather information from NOAA ADDS.
func (p Provider) GetWeather() (*weather.Weather, error) {
	response, err := http.Get(p.url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode >= 500 {
		// The METAR server occasionally times out and throws 5xx
		// errors. Don't treat these as an error - just keep using
		// the last weather report, and try again later.
		return nil, nil
	} else if response.StatusCode != 200 {
		err = fmt.Errorf("Could not fetch METAR: %s", response.Status)
		return nil, err
	}

	var resp addsResponse
	err = xml.NewDecoder(response.Body).Decode(&resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Metars) != 1 {
		err = fmt.Errorf("Expected one METAR in response body, got %d", len(resp.Metars))
		return nil, err
	}

	m := resp.Metars[0]

	updated, err := time.Parse(time.RFC3339, m.ObservationTime)
	if err != nil {
		return nil, err
	}

	return &weather.Weather{
		Location:    m.StationID,
		Condition:   m.getCondition(),
		Description: m.encodeMetar(p.stripRemarks, p.includeFlightCat),
		Temperature: unit.FromCelsius(m.Temperature),
		Humidity:    relativeHumidity(m.Temperature, m.Dewpoint),
		Pressure:    m.getBarometricPressure(),
		Wind: weather.Wind{
			Speed:     unit.Speed(float64(m.WindSpeed)) * unit.Knot,
			Direction: weather.Direction(m.WindDirection),
		},
		CloudCover:  m.getCloudCover(),
		Updated:     updated,
		Attribution: "NWS",
	}, nil
}
