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

	"github.com/martinlindhe/unit"
	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
)

func shouldReturn(temps ...string) {
	for zoneIndex, temp := range temps {
		tempFile := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", zoneIndex)
		afero.WriteFile(fs, tempFile, []byte(temp), 0644)
	}
}

func TestCputemp(t *testing.T) {
	fs = afero.NewMemMapFs()
	testBar.New(t)

	shouldReturn("48800", "22200")

	temp0 := DefaultZone()

	temp1 := Zone("thermal_zone1").
		OutputTemplate(outputs.TextTemplate(`{{.Fahrenheit | printf "%.0f"}}`))

	temp2 := Zone("thermal_zone2").
		OutputTemplate(outputs.TextTemplate(`{{.Kelvin | printf "%.0f"}}`))

	testBar.Run(temp0, temp1, temp2)

	out := testBar.LatestOutput()
	out.At(0).AssertText("48.8℃", "on start")
	out.At(1).AssertText("72", "on start")
	out.At(2).AssertError("on start with invalid zone")

	shouldReturn("42123", "20000")
	testBar.AssertNoOutput("until refresh")
	testBar.Tick()

	out = testBar.LatestOutput()
	out.At(0).AssertText("42.1℃", "on next tick")

	temp0.UrgentWhen(func(t unit.Temperature) bool { return t.Celsius() > 30 })
	out = testBar.LatestOutput()
	urgent, _ := out.At(0).Segment().IsUrgent()
	assert.True(t, urgent, "on urgent func change")

	red := bar.Color("red")
	green := bar.Color("green")
	temp1.OutputColor(func(t unit.Temperature) bar.Color {
		if t.Celsius() > 20 {
			return red
		}
		return green
	})
	out = testBar.LatestOutput()
	col, _ := out.At(1).Segment().GetColor()
	assert.Equal(t, green, col, "on color func change")

	temp2.OutputTemplate(outputs.TextTemplate(`{{.Kelvin | printf "%.0f"}} kelvin`))
	testBar.AssertNoOutput("on error'd template change")
	testBar.Click(2)
	testBar.LatestOutput().At(2).AssertError("error persists at restart")

	shouldReturn("22222", "22222")
	testBar.Tick()

	testBar.LatestOutput().AssertEqual(
		outputs.Group(
			outputs.Text("22.2℃").Urgent(false),
			outputs.Text("72").Color(red),
			outputs.Errorf("open /sys/class/thermal/thermal_zone2/temp: file does not exist"),
		),
		"on next tick")

	temp2.RefreshInterval(time.Second)
	testBar.AssertNoOutput("on refresh interval change")

	shouldReturn("0", "0", "0")
	testBar.Click(2)
	testBar.Tick()
	// Only temp2 has an update, since temp0 and temp1 are still
	// on the 3 second refresh interval.
	testBar.LatestOutput().At(2).AssertText(
		"273 kelvin",
		"on next tick when zone becomes available")

	shouldReturn("0", "0", "invalid")
	testBar.Tick()
	testBar.LatestOutput().At(2).AssertError("On invalid numeric value")
	testBar.AssertNoOutput("until tick")
}
