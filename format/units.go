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

// Package format provides utility methods for formatting units.
package format // import "barista.run/format"

import (
	"fmt"
	"math"
	"strings"
	"time"

	"barista.run/base/value"
	"github.com/martinlindhe/unit"
)

var suffixesSI = []string{
	"y", "z", "a", "f", "p", "n", "µ", "m",
	"",
	"k", "M", "G", "T", "P", "E", "Z", "Y",
}

// Value represents a formatted Unit value.
type Value struct {
	number string
	Unit   string
}

// Number returns a representation that occupies at least `width` characters,
// increasing precision to fill the available space.
func (v Value) Number(width int) string {
	minWidth := strings.IndexRune(v.number, '.')
	if minWidth == 0 {
		minWidth = strings.IndexFunc(v.number, func(r rune) bool {
			return r != '.' && r != '0'
		})
		if minWidth >= 0 {
			minWidth++
		}
	}
	if minWidth < 0 {
		minWidth = len(v.number)
	}
	if width < minWidth {
		width = minWidth
	}
	if width > len(v.number) {
		return strings.Repeat(" ", width-len(v.number)) + v.number
	}
	out := v.number[:width]
	if out[width-1] == '.' {
		out = " " + out[:width-1]
	}
	return out
}

// String formats the value as a string
func (v Value) String() string {
	return v.StringW(1)
}

// StringW formats the value as a string, using the given width for the numeric
// portion.
func (v Value) StringW(w int) string {
	return v.Number(w) + v.Unit
}

// Values represents an ordered list of values, e.g. hours/minutes/seconds.
type Values []Value

// String formats the values as a string
func (v Values) String() string {
	r := ""
	w := 1
	if len(v) == 1 {
		w += 3
	}
	for _, val := range v {
		r += val.StringW(w)
	}
	return r
}

func pow1000(n int) float64 {
	return math.Pow10(n * 3)
}

func val(n float64, unit string) Value {
	if n < 0 {
		v := val(-n, unit)
		v.number = "-" + v.number
		return v
	}
	epsilon := math.Nextafter(0.0, n)
	if n <= epsilon {
		return Value{"0", unit}
	}
	valStr := fmt.Sprintf("%.7f", n)
	valStr = strings.TrimLeft(valStr, "0")
	return Value{valStr, unit}
}

// SI formats an SI unit value by scaling it to a sensible multiplier, and
// returns a three-character value (four if negative) and a suffix that's either
// empty or a single character.
// e.g. format.SI((20480*unit.Megabyte).Bytes(), "B") == {"20.48000", "GB"}
// or format.SI((0.001234*unit.Foot).Meter(), "m") == {"376.1232", "µm"}
func SI(v float64, unit string) Value {
	if v < 0 {
		inv := SI(-v, unit)
		inv.number = "-" + inv.number
		return inv
	}
	epsilon := math.Nextafter(0.0, v)
	if v <= epsilon {
		return Value{"0", unit}
	}
	f := pow1000(-8)
	if v < f {
		return val(v/f, suffixesSI[0]+unit)
	}
	for i, s := range suffixesSI {
		next := pow1000(i - 7)
		if v < next {
			return val(v/f, s+unit)
		}
		f = next
	}
	f = pow1000(len(suffixesSI) - 8 - 1)
	return val(v/f, suffixesSI[len(suffixesSI)-1]+unit)
}

type temperatureUnit int

// Pass in these constants to SetTemperatureUnit to control the default format.
const (
	Celsius temperatureUnit = iota
	Fahrenheit
	Kelvin
)

var defaultTempUnit value.Value

// SetTemperatureUnit sets the default unit used when formatting temperatures.
func SetTemperatureUnit(f temperatureUnit) {
	defaultTempUnit.Set(f)
}

//go:generate ruby siunit.rb

// Unit formats a unit.Unit value to the most appropriately scaled base unit.
// For example, Unit(length) is equivalent to SI(length.Meters(), "m").
// For non-base units (e.g. feet), use SI(length.Feet(), "ft").
func Unit(value interface{}) (Values, bool) {
	if fVal, ok := SIUnit(value); ok {
		return Values{fVal}, ok
	}
	switch v := value.(type) {
	case unit.Unit:
		return Values{SI(float64(v), "")}, true
	case unit.Duration:
		return Duration(
			time.Duration(v.Nanoseconds()) * time.Nanosecond), true
	case time.Duration:
		return Duration(v), true
	case unit.Temperature:
		u, _ := defaultTempUnit.Get().(temperatureUnit)
		switch u {
		case Fahrenheit:
			return Values{val(v.Fahrenheit(), "℉")}, true
		case Kelvin:
			return Values{val(v.Kelvin(), "K")}, true
		default:
			return Values{val(v.Celsius(), "℃")}, true
		}
	}
	return nil, false
}

// Duration formats a time.Duration by providing the two most significant units.
func Duration(d time.Duration) Values {
	if d.Hours() >= 24 {
		return Values{
			val(float64(int(d.Hours()))/24.0, "d"),
			val(float64(int(d.Hours())%24), "h"),
		}
	}
	if d.Minutes() >= 60 {
		return Values{
			val(d.Truncate(time.Hour).Hours(), "h"),
			val(float64(int(d.Minutes())%60), "m"),
		}
	}
	if d.Seconds() >= 60 {
		return Values{
			val(d.Truncate(time.Minute).Minutes(), "m"),
			val(float64(int(d.Seconds())%60), "s"),
		}
	}
	return Values{SI(d.Seconds(), "s")}
}
