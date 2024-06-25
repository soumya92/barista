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

package metar

import (
	"testing"
	"time"

	"github.com/soumya92/barista/modules/weather"
	testServer "github.com/soumya92/barista/testing/httpserver"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	ts := testServer.New()
	defer ts.Close()

	provider := Station("KBFI").StripRemarks().IncludeFlightCat().Build().(*provider)
	provider.url = ts.URL + "/static/example.xml"

	wthr, err := provider.GetWeather()
	require.NoError(t, err)
	require.NotNil(t, wthr)
	require.Equal(t, weather.Weather{
		Location:    "KBFI",
		Condition:   weather.Overcast,
		Description: "[VFR] KBFI 252053Z 00000KT 10SM BKN090 OVC110 09.4/M03.3 A3013",
		Humidity:    0.4070166689067204,
		Pressure:    1020.4 * unit.Millibar,
		Temperature: unit.FromCelsius(9.4),
		CloudCover:  1.0,
		Updated:     time.Unix(1543179180, 0).In(time.UTC),
		Attribution: "NWS",
	}, wthr)

	provider.url = ts.URL + "/code/503"
	wthr, err = provider.GetWeather()
	require.NoError(t, err)
	require.NotNil(t, wthr)
	require.Equal(t, weather.Weather{
		Location:    "KBFI",
		Condition:   weather.Overcast,
		Description: "[VFR] KBFI 252053Z 00000KT 10SM BKN090 OVC110 09.4/M03.3 A3013",
		Humidity:    0.4070166689067204,
		Pressure:    1020.4 * unit.Millibar,
		Temperature: unit.FromCelsius(9.4),
		CloudCover:  1.0,
		Updated:     time.Unix(1543179180, 0).In(time.UTC),
		Attribution: "NWS",
	}, wthr)

	provider.url = ts.URL + "/code/401"
	wthr, err = provider.GetWeather()
	require.Error(t, err)

	provider.url = ts.URL + "/static/bad.xml"
	wthr, err = provider.GetWeather()
	require.Error(t, err)

	provider.url = ts.URL + "/static/no-entries.xml"
	wthr, err = provider.GetWeather()
	require.Error(t, err)
}
