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

package format

import (
	"go/importer"
	"go/types"
	"testing"
	"time"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"
)

func TestSI(t *testing.T) {
	assertEqual := func(expected string, formatted Value, width int) {
		require.Equal(t, expected, formatted.Number(width)+" "+formatted.Unit, "%s %s != %s (%v)",
			formatted.Number(width), formatted.Unit, expected, formatted)
	}

	assertEqual(" 10 km", SI((10*unit.Kilometer).Meters(), "m"), 3)
	assertEqual("123.0 G", SI((123*unit.Gigapascal).Pascals(), ""), 5)
	assertEqual("4 µW", SI((4444*unit.Nanowatt).Watts(), "W"), 0)
	assertEqual(" 1 MHz", SI((1048576*unit.Hertz).Hertz(), "Hz"), 2)
	assertEqual("  0 ", SI(0, ""), 3)
	assertEqual(" 0 k", SI(0, "k"), 2)
	assertEqual("    0 ", SI(0.0, ""), 5)
	assertEqual(" 111 mg", SI((0.1111*unit.Gram).Grams(), "g"), 4)
	assertEqual(".04 yJ", SI((0.044*unit.Yoctojoule).Joules(), "J"), 2)
	assertEqual("1234 YB", SI((1234*unit.Yottabyte).Bytes(), "B"), 2)
	assertEqual("8.0 Tb/s", SI(unit.TerabytePerSecond.BitsPerSecond(), "b/s"), 3)

	// This should be rare in practice, the only reason negative distances make
	// sense is if they're combined with some sort of reference point.
	assertEqual(" -10 mm", SI((-10*unit.Millimeter).Meters(), "m"), 4)
}

func TestSIUnits(t *testing.T) {
	assertFormatted := func(expected string, unit interface{}, width int) {
		formatted, ok := Unit(unit)
		require.True(t, ok, "could not format %v as a unit (expected %s)", unit, expected)
		require.Equal(t, expected, formatted[0].String(),
			"%s != %s (%v)", formatted[0].String(), expected, unit)
	}

	assertFormatted("10km", 10*unit.Kilometer, 3)
	assertFormatted("123GPa", 123*unit.Gigapascal, 5)
	assertFormatted("4µW", 4444*unit.Nanowatt, 0)
	assertFormatted("1MHz", 1048576*unit.Hertz, 2)
	assertFormatted("0A", 0*unit.Ampere, 3)
	assertFormatted("0Ω", 0*unit.Ohm, 2)
	assertFormatted("0N", 0.0*unit.Newton, 5)
	assertFormatted("111mg", 0.1111*unit.Gram, 4)
	assertFormatted(".04yJ", 0.044*unit.Yoctojoule, 3)
	assertFormatted("1234YB", 1234*unit.Yottabyte, 2)
	assertFormatted("1TB/s", unit.TerabytePerSecond, 3)
	assertFormatted("4k", unit.Unit(4000), 2)

	assertFormatted(".001℃", unit.FromCelsius(0.001), 4)
	SetTemperatureUnit(Fahrenheit)
	assertFormatted("-3℉", unit.FromFahrenheit(-3), 3)
	SetTemperatureUnit(Kelvin)
	assertFormatted("255K", unit.FromFahrenheit(0), 1)

	// This should be rare in practice, the only reason negative distances make
	// sense is if they're combined with some sort of reference point.
	assertFormatted("-10mm", -10*unit.Millimeter, 4)
}

func TestDurations(t *testing.T) {
	for _, tc := range []struct {
		expected string
		actual   interface{}
	}{
		{"1h0m", time.Hour},
		{"2h1m", 2*time.Hour + time.Minute},
		{"1d0h", 24*time.Hour + time.Minute},
		{"32.5s", unit.Duration(32.5) * unit.Second},
		{"32.5s", 32*time.Second + 500*time.Millisecond},
		{"1m0s", time.Minute + time.Millisecond},
	} {
		out, ok := Unit(tc.actual)
		require.True(t, ok, "Could not format %v as Unit", tc.actual)
		require.Equal(t, tc.expected, out.String())
	}
}

func TestAllUnitsHandled(t *testing.T) {
	// These units are handled outside the SIUnits function.
	siUnitsHandled["Unit"] = unit.Unit(1)
	siUnitsHandled["Temperature"] = unit.FromFahrenheit(72)
	siUnitsHandled["Duration"] = unit.Duration(1)

	// Verify that all units supported by the unit package are included in the
	// test cases.
	pkg, err := importer.For("source", nil).Import("github.com/martinlindhe/unit")
	require.NoError(t, err)
	for _, declName := range pkg.Scope().Names() {
		obj := pkg.Scope().Lookup(declName)
		if typ, ok := obj.(*types.TypeName); ok {
			require.Contains(t, siUnitsHandled, typ.Name(),
				"unhandled 'unit.%s'", typ.Name())
		}
	}

	// Test that all units can be formatted.
	for unit, value := range siUnitsHandled {
		f, ok := Unit(value)
		require.True(t, ok, "Cannot handle unit.%s value %v", unit, value)
		require.NotEmpty(t, f.String())
	}

	// Test unhandled units.
	_, ok := Unit(4)
	require.False(t, ok, "Raw number not handled")

	_, ok = Unit(struct{}{})
	require.False(t, ok, "struct not handled in Unit")

	_, ok = Unit("foobar")
	require.False(t, ok, "string not handled in Unit")
}
