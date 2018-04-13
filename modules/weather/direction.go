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

// Deg returns the direction in meteorological degrees.
func (d Direction) Deg() int {
	return int(d)
}

// Cardinal returns the cardinal direction.
func (d Direction) Cardinal() string {
	cardinal := ""
	deg := d.Deg()
	m := 34 // rounded from (90/4 + 90/8)
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
