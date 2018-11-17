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
	"testing"

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
	assertEqual("0.0 ", SI(0, ""), 3)
	assertEqual(" 0 k", SI(0, "k"), 2)
	assertEqual("  0.0 ", SI(0.0, ""), 5)
	assertEqual(" 111 mg", SI((0.1111*unit.Gram).Grams(), "g"), 4)
	assertEqual(".04 yJ", SI((0.044*unit.Yoctojoule).Joules(), "J"), 3)
	assertEqual("1234 YB", SI((1234*unit.Yottabyte).Bytes(), "B"), 2)
	assertEqual("8.0 Tb/s", SI(unit.TerabytePerSecond.BitsPerSecond(), "b/s"), 3)

	// This should be rare in practice, the only reason negative distances make
	// sense is if they're combined with some sort of reference point.
	assertEqual(" -10 mm", SI((-10*unit.Millimeter).Meters(), "m"), 4)
}
