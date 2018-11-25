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

package cputemp

import (
	"fmt"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"

	"github.com/martinlindhe/unit"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func setTypes(types ...string) {
	for zoneIndex, typ := range types {
		tempFile := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/type", zoneIndex)
		afero.WriteFile(fs, tempFile, []byte(typ), 0644)
	}
}

func shouldReturn(temps ...string) {
	for zoneIndex, temp := range temps {
		tempFile := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", zoneIndex)
		afero.WriteFile(fs, tempFile, []byte(temp), 0644)
	}
}

func TestCputemp(t *testing.T) {
	fs = afero.NewMemMapFs()
	testBar.New(t)

	setTypes("x86_pkg_temp")
	shouldReturn("48800", "22200")

	temp0 := New()
	temp1 := Zone("thermal_zone1").Output(func(t unit.Temperature) bar.Output {
		return outputs.Textf("%.0f", t.Fahrenheit())
	})
	temp2 := Zone("thermal_zone2").Output(func(t unit.Temperature) bar.Output {
		return outputs.Textf("%.0f", t.Kelvin())
	})

	testBar.Run(temp0, temp1, temp2)
	testBar.NextOutput("first module")
	testBar.NextOutput("second module")
	testBar.NextOutput("third module")

	out := testBar.NextOutput("on start, error handlers setup")
	out.At(0).AssertText("48.8℃", "on start")
	out.At(1).AssertText("72", "on start")
	out.At(2).AssertError("on start with invalid zone")

	shouldReturn("42123", "20000")
	testBar.AssertNoOutput("until refresh")
	testBar.Tick()

	out = testBar.LatestOutput(0, 1)
	out.At(0).AssertText("42.1℃", "on next tick")

	temp2.Output(func(t unit.Temperature) bar.Output {
		return outputs.Textf("%.0f kelvin", t.Kelvin())
	})
	testBar.AssertNoOutput("on error'd template change")
	out.At(2).LeftClick()
	testBar.NextOutput("with error gone").Expect()
	testBar.LatestOutput(2).At(2).AssertError("error persists at restart")
	testBar.LatestOutput(2).Expect("sets restart click handler")

	shouldReturn("22222", "22222")
	testBar.Tick()

	out = testBar.LatestOutput(0, 1)
	out.At(0).AssertEqual(outputs.Text("22.2℃"))
	out.At(1).AssertEqual(outputs.Text("72"))
	errStr := out.At(2).AssertError()
	require.Contains(t, errStr, "file does not exist")

	temp2.RefreshInterval(time.Second)
	testBar.AssertNoOutput("on refresh interval change")

	shouldReturn("0", "0", "0")
	out.At(2).LeftClick()
	// TODO: cleanup.
	testBar.LatestOutput(2).Expect(
		"after restart, to clear error segment")
	testBar.LatestOutput(2).Expect(
		"after restart, because of interval change")
	testBar.LatestOutput(2).Expect(
		"after restart, because of format change")
	testBar.Tick()
	// Only temp2 has an update, since temp0 and temp1 are still
	// on the 3 second refresh interval.
	testBar.LatestOutput(2).At(2).AssertText(
		"273 kelvin",
		"on next tick when zone becomes available")

	shouldReturn("0", "0", "invalid")
	testBar.Tick()
	// 0 and 1 are unchanged, so only 2 should update.
	testBar.LatestOutput(2).At(2).AssertError("On invalid numeric value")
	testBar.LatestOutput(2).Expect("set click handler to restart module")
	testBar.AssertNoOutput("until tick")
}

func TestDefaultZoneDetection(t *testing.T) {
	fs = afero.NewMemMapFs()
	testBar.New(t)

	setTypes("acpitz", "iwlwifi")
	shouldReturn("0", "0")

	tempDefault := New()
	tempWifi := OfType("iwlwifi")
	testBar.Run(tempDefault, tempWifi)
	out := testBar.LatestOutput()
	out.At(0).AssertError("no x86_pkg_temp")
	out.At(1).AssertText("0.0℃")

	testBar.New(t)

	setTypes("acpitz", "iwlwifi", "x86_pkg_temp")
	shouldReturn("0", "0")

	tempDefault = New()
	tempNotFound := OfType("not_found")
	testBar.Run(tempDefault, tempNotFound)
	out = testBar.LatestOutput()
	out.At(0).AssertError("temperature missing")
	out.At(1).AssertError("no zone of type")
}
