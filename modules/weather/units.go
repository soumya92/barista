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

// K returns the temperature in kelvin.
func (t Temperature) K() int {
	return int(t)
}

// TemperatureFromK creates a kelvin temperature value.
func TemperatureFromK(k float64) Temperature {
	return Temperature(k)
}

// C returns the temperature in degrees celcius.
func (t Temperature) C() int {
	return int(float64(t) - 273.15)
}

// TemperatureFromC creates a celcius temperature value.
func TemperatureFromC(c float64) Temperature {
	return Temperature(c + 273.15)
}

// F returns the temperature in degrees fahrenheit.
func (t Temperature) F() int {
	c := float64(t) - 273.15
	return int(c*1.8 + 32)
}

// TemperatureFromF creates a fahrenheit temperature value.
func TemperatureFromF(f float64) Temperature {
	return TemperatureFromC((f - 32.0) / 1.8)
}

// Millibar returns pressure in millibars (hPa).
func (p Pressure) Millibar() float64 {
	return float64(p)
}

// PressureFromMillibar creates a millibar pressure value.
func PressureFromMillibar(mb float64) Pressure {
	return Pressure(mb)
}

// Pascal returns pressure in pascals.
func (p Pressure) Pascal() float64 {
	return p.Millibar() * 100
}

// PressureFromPascal creates a millibar pressure value.
func PressureFromPascal(pa float64) Pressure {
	return Pressure(pa * 0.01)
}

// Atm returns pressure in atmospheres.
func (p Pressure) Atm() float64 {
	return p.Millibar() * 0.000986923
}

// PressureFromAtm creates an atmospheric pressure value.
func PressureFromAtm(atm float64) Pressure {
	return Pressure(atm * 1013.25)
}

// Torr returns pressure in torr. ~= mmHg.
func (p Pressure) Torr() float64 {
	return p.Millibar() * 0.750062
}

// PressureFromTorr creates a torr pressure value.
func PressureFromTorr(t float64) Pressure {
	return Pressure(t * 1.33322)
}

// Psi returns pressure in pounds per square inch.
func (p Pressure) Psi() float64 {
	return p.Millibar() * 0.01450377
}

// PressureFromPsi creates a pounds/sq. in. pressure value.
func PressureFromPsi(psi float64) Pressure {
	return Pressure(psi * 68.9476)
}

// Ms returns the speed in meters per second.
func (s Speed) Ms() float64 {
	return float64(s)
}

// SpeedFromMs creates a meters/second speed value.
func SpeedFromMs(ms float64) Speed {
	return Speed(ms)
}

// Kmh returns the speed in kilometers per hour.
func (s Speed) Kmh() float64 {
	return s.Ms() * 3.6
}

// SpeedFromKmh creates a kilometers/hour speed value.
func SpeedFromKmh(kmh float64) Speed {
	return Speed(kmh / 3.6)
}

// Mph returns the speed in miles per hour.
func (s Speed) Mph() float64 {
	return s.Ms() * 2.23694
}

// SpeedFromMph creates a miles/hour speed value.
func SpeedFromMph(mph float64) Speed {
	return Speed(mph / 2.23694)
}

// Knots returns the speed in knots.
func (s Speed) Knots() float64 {
	return s.Ms() * 1.94384
}

// SpeedFromKnots creates a knots speed value.
func SpeedFromKnots(kts float64) Speed {
	return Speed(kts / 1.94384)
}

// Deg returns the direction in meteorological degrees.
func (d Direction) Deg() int {
	return int(d)
}

// Cardinal returns the cardinal direction.
func (d Direction) Cardinal() string {
	cardinal := ""
	deg := d.Deg()
	m := 34
	// primary cardinal direction first. N, E, S, W.
	switch {
	case deg < m || deg > 360-m:
		cardinal = "N"
	case 90-m < deg && deg < 90+m:
		cardinal = "E"
	case 180-m < deg && deg < 180+m:
		cardinal = "S"
	case 270-m < deg && deg < 270+m:
		cardinal = "W"
	}
	// Now append the midway points. NE, NW, SE, SW.
	switch {
	case 45-m < deg && deg < 45+m:
		cardinal += "NE"
	case 135-m < deg && deg < 135+m:
		cardinal += "SE"
	case 225-m < deg && deg < 225+m:
		cardinal += "SW"
	case 315-m < deg && deg < 315+m:
		cardinal += "NW"
	}
	return cardinal
}
