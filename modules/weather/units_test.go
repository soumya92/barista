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

package weather

import (
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func TestTemperature(t *testing.T) {
	a0 := TemperatureFromK(0)
	assert.Equal(t, 0, a0.K())
	assert.InDelta(t, -273, a0.C(), 1)
	assert.InDelta(t, -460, a0.F(), 1)

	freezing := TemperatureFromC(0)
	assert.InDelta(t, 273, freezing.K(), 1)
	assert.Equal(t, 0, freezing.C())
	assert.InDelta(t, 32, freezing.F(), 1)

	boiling := TemperatureFromF(212)
	assert.InDelta(t, 373, boiling.K(), 1)
	assert.InDelta(t, 100, boiling.C(), 1)
	assert.Equal(t, 212, boiling.F())
}

func TestPressure(t *testing.T) {
	p := PressureFromMillibar(1000)
	assert.InDelta(t, 1000, p.Millibar(), 1)
	p = PressureFromPascal(p.Pascal())
	assert.InDelta(t, 1000, p.Millibar(), 1)
	p = PressureFromAtm(p.Atm())
	assert.InDelta(t, 1000, p.Millibar(), 1)
	p = PressureFromTorr(p.Torr())
	assert.InDelta(t, 1000, p.Millibar(), 1)
	p = PressureFromInHg(p.InHg())
	assert.InDelta(t, 1000, p.Millibar(), 1)
	p = PressureFromPsi(p.Psi())
	assert.InDelta(t, 1000, p.Millibar(), 1)
}

func TestSpeed(t *testing.T) {
	s := SpeedFromMs(1000)
	assert.InDelta(t, 1000, s.Ms(), 1)
	s = SpeedFromKmh(s.Kmh())
	assert.InDelta(t, 1000, s.Ms(), 1)
	s = SpeedFromMph(s.Mph())
	assert.InDelta(t, 1000, s.Ms(), 1)
	s = SpeedFromKnots(s.Knots())
	assert.InDelta(t, 1000, s.Ms(), 1)
}

func TestDirection(t *testing.T) {
	for _, c := range []struct {
		deg  int
		card string
	}{
		// Basic sanity checks.
		{0, "N"}, {90, "E"}, {180, "S"}, {270, "W"},
		{45, "NE"}, {135, "SE"}, {225, "SW"}, {315, "NW"},

		// All boundary conditions. Should be safe to assume that
		// points within are correct.
		{349, "N"}, {11, "N"},
		{12, "NNE"}, {33, "NNE"},
		{34, "NE"}, {56, "NE"},
		{57, "ENE"}, {78, "ENE"},
		{79, "E"}, {101, "E"},
		{102, "ESE"}, {123, "ESE"},
		{124, "SE"}, {146, "SE"},
		{147, "SSE"}, {168, "SSE"},
		{169, "S"}, {191, "S"},
		{192, "SSW"}, {213, "SSW"},
		{214, "SW"}, {236, "SW"},
		{237, "WSW"}, {258, "WSW"},
		{259, "W"}, {281, "W"},
		{282, "WNW"}, {303, "WNW"},
		{304, "NW"}, {326, "NW"},
		{327, "NNW"}, {348, "NNW"},
	} {
		dir := Direction(c.deg)
		assert.Equal(t, c.card, dir.Cardinal())
		assert.Equal(t, c.deg, dir.Deg())
	}
}
