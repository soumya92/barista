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

	"github.com/stretchr/testify/require"
)

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
		require.Equal(t, c.card, dir.Cardinal())
		require.Equal(t, c.deg, dir.Deg())
	}
}
