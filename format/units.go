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

// Number returns a representaion that occupies at least `width` characters,
// increasing precision to fill the available space.
func (v Value) Number(width int) string {
	minWidth := strings.IndexRune(v.number, '.')
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

// SI formats an SI unit value by scaling it to a sensible multiplier, and
// returns a three-character value (four if negative) and a suffix that's either
// empty or a single character.
// e.g. format.SI((20480*unit.Megabyte).Bytes(), "B") == {"20.48000", "GB"}
// or format.SI((0.001234*unit.Foot).Meter(), "m") == {"376.1232", "µm"}
func SI(val float64, unit string) Value {
	if val < 0 {
		inv := SI(-val, unit)
		inv.number = "-" + inv.number
		return inv
	}
	f := 1e-24
	epsilon := math.Nextafter(0.0, val)
	if val <= epsilon {
		return Value{"0.0", unit}
	}
	if val < f {
		return Value{fmt.Sprintf("%.7f", val/f)[1:], suffixesSI[0] + unit}
	}
	for _, s := range suffixesSI {
		next := f * 1e3
		if val < next {
			return Value{fmt.Sprintf("%.7f", val/f), s + unit}
		}
		f = next
	}
	f = f / 1e3
	return Value{fmt.Sprintf("%.f", val/f), suffixesSI[len(suffixesSI)-1] + unit}
}
