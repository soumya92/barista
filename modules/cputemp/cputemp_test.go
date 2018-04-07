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

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

type zones map[string]float64

func shouldReturn(temps ...string) {
	for zoneIndex, temp := range temps {
		tempFile := fmt.Sprintf("/sys/class/thermal/thermal_zone%d/temp", zoneIndex)
		afero.WriteFile(fs, tempFile, []byte(temp), 0644)
	}
}

func TestCputemp(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()
	scheduler.TestMode(true)

	shouldReturn("48800", "22200")

	temp0 := DefaultZone().OutputTemplate(outputs.TextTemplate(`{{.C}}`))
	tester0 := testModule.NewOutputTester(t, temp0)

	temp1 := Zone("thermal_zone1").
		OutputTemplate(outputs.TextTemplate(`{{.F}}`))
	tester1 := testModule.NewOutputTester(t, temp1)

	temp2 := Zone("thermal_zone2").
		OutputTemplate(outputs.TextTemplate(`{{.K}}`))
	tester2 := testModule.NewOutputTester(t, temp2)

	out := tester0.AssertOutput("on start")
	assert.Equal(outputs.Text("49").Segments(), out)

	out = tester1.AssertOutput("on start")
	assert.Equal(outputs.Text("72").Segments(), out)

	tester2.AssertError("on start with invalid zone")

	shouldReturn("42123", "20000")

	tester1.AssertNoOutput("until refresh")
	tester2.AssertNoOutput("until refresh")

	scheduler.NextTick()

	out = tester0.AssertOutput("on next tick")
	assert.Equal(outputs.Text("42").Segments(), out)
	tester1.AssertOutput("on next tick")
	tester2.AssertError("on each tick")

	temp0.UrgentWhen(func(t Temperature) bool { return t.C() > 30 })
	out = tester0.AssertOutput("on urgent func change")
	assert.Equal(outputs.Text("42").Urgent(true).Segments(), out)

	red := bar.Color("red")
	green := bar.Color("green")
	temp1.OutputColor(func(t Temperature) bar.Color {
		if t.C() > 20 {
			return red
		}
		return green
	})
	out = tester1.AssertOutput("on color func change")
	assert.Equal(outputs.Text("68").Color(green).Segments(), out)

	temp2.OutputTemplate(outputs.TextTemplate(`{{.K}} kelvin`))
	tester2.AssertError("error persists even with template change")

	shouldReturn("22222", "22222")
	scheduler.NextTick()

	out = tester0.AssertOutput("on next tick")
	assert.Equal(outputs.Text("22").Urgent(false).Segments(), out)

	out = tester1.AssertOutput("on next tick")
	assert.Equal(outputs.Text("72").Color(red).Segments(), out)

	tester2.AssertError("on each tick")

	temp2.RefreshInterval(time.Second)
	tester2.AssertNoOutput("on refresh interval change")

	shouldReturn("0", "0", "0")
	scheduler.NextTick()
	// Only temp2 has an update, since temp0 and temp1 are still
	// on the 3 second refresh interval.
	out = tester2.AssertOutput("on next tick when zone becomes available")
	assert.Equal(outputs.Text("273 kelvin").Segments(), out)

	shouldReturn("0", "0", "invalid")
	scheduler.NextTick()
	tester2.AssertError("On invalid numeric value")

	tester0.AssertNoOutput("until tick")
	tester1.AssertNoOutput("until tick")
	tester2.AssertNoOutput("until tick")
}
