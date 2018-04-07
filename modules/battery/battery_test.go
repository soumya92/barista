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

package battery

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

type battery map[string]interface{}

func write(battery battery) {
	var buffer bytes.Buffer
	for key, value := range battery {
		buffer.WriteString("POWER_SUPPLY_")
		buffer.WriteString(key)
		buffer.WriteString("=")
		buffer.WriteString(fmt.Sprintf("%v", value))
		buffer.WriteString("\n")
	}
	batteryFile := batteryPath(battery["NAME"].(string))
	afero.WriteFile(fs, batteryFile, buffer.Bytes(), 0644)
}

func TestUnknownAndDisconnected(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()

	// No battery.
	info := batteryInfo("BAT0")
	assert.Equal("Disconnected", info.Status)

	// Unknown status.
	write(battery{"NAME": "BAT1"})
	info = batteryInfo("BAT1")
	assert.Equal("Unknown", info.Status)

	write(battery{
		"NAME":   "BAT2",
		"STATUS": "Unknown",
	})
	info = batteryInfo("BAT2")
	assert.Equal("Unknown", info.Status)

	// Make sure nothing panics when values are missing.
	assert.InDelta(0, info.Remaining(), 0.000001)
	assert.Equal(0, info.RemainingPct())
	assert.Equal(time.Duration(0), info.RemainingTime())
}

func TestGarbageFiles(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()

	afero.WriteFile(fs, batteryPath("BAT0"), []byte(`
POWER_SUPPLY_NAME=BAT0
POWER_SUPPLY_STATUS=Charging
This line is weird and should be ignored
POWER_SUPPLY_VOLTAGE_NOW=12000000
POWER_SUPPLY_CURRENT_NOW=500000
POWER_SUPPLY_ENERGY_FULL=60000000
POWER_SUPPLY_CHARGE_NOW=3000000
POWER_SUPPLY_CHARGE_FULL_DESIGN=xyzabc
POWER_SUPPLY_TECHNOLOGY=NiCd
POWER_SUPPLY_TECHNOLOGY=malformed=line
And an empty line follows

`), 0644)
	info := batteryInfo("BAT0")

	assert.Equal("Charging", info.Status)
	assert.InDelta(12, info.Voltage, 0.01)
	assert.InDelta(6, info.Power, 0.01)
	assert.InDelta(60, info.EnergyFull, 0.01)
	assert.InDelta(36, info.EnergyNow, 0.01)
	// invalid entry is not parsed.
	assert.Equal(0.0, info.EnergyMax)
	// invalid entry does not overwrite previous.
	assert.Equal("NiCd", info.Technology)
}

var micros = 1000 * 1000

func TestSimple(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()
	write(battery{
		"NAME":               "BAT0",
		"STATUS":             "Charging",
		"PRESENT":            1,
		"TECHNOLOGY":         "Li-poly",
		"VOLTAGE_NOW":        20 * micros,
		"POWER_NOW":          10 * micros,
		"ENERGY_FULL_DESIGN": 50 * micros,
		"ENERGY_FULL":        40 * micros,
		"ENERGY_NOW":         20 * micros,
		"CAPACITY":           50,
	})

	write(battery{
		"NAME":               "BAT1",
		"STATUS":             "Full",
		"PRESENT":            1,
		"VOLTAGE_NOW":        20 * micros,
		"POWER_NOW":          10 * micros,
		"ENERGY_FULL_DESIGN": 50 * micros,
		"ENERGY_FULL":        40 * micros,
		"ENERGY_NOW":         40 * micros,
		"CAPACITY":           100,
	})

	write(battery{
		"NAME":               "BAT2",
		"STATUS":             "Discharging",
		"PRESENT":            1,
		"TECHNOLOGY":         "NiCd",
		"VOLTAGE_NOW":        20 * micros,
		"CURRENT_NOW":        int(0.5 * float64(micros)),
		"CHARGE_FULL_DESIGN": 5 * micros,
		"CHARGE_FULL":        int(4.4 * float64(micros)),
		"CHARGE_NOW":         int(2.2 * float64(micros)),
		"CAPACITY":           50,
	})

	info := batteryInfo("BAT0")
	assert.Equal("Charging", info.Status)
	assert.Equal("Li-poly", info.Technology)
	assert.InDelta(20.0, info.Voltage, 0.01)
	assert.InDelta(10.0, info.Power, 0.01)
	assert.InDelta(0.5, info.Remaining(), 0.0001)
	assert.InDelta(20.0, info.EnergyNow, 0.01)
	assert.InDelta(40.0, info.EnergyFull, 0.01)
	assert.InDelta(50.0, info.EnergyMax, 0.01)
	assert.Equal(2*time.Hour, info.RemainingTime())
	assert.Equal(50, info.Capacity)
	assert.True(info.PluggedIn())

	info = batteryInfo("BAT1")
	assert.Equal("Full", info.Status)
	assert.True(info.PluggedIn())

	info = batteryInfo("BAT2")
	assert.InDelta(20.0, info.Voltage, 0.01)
	assert.InDelta(100.0, info.EnergyMax, 0.01)
	assert.InDelta(88.0, info.EnergyFull, 0.01)
	assert.InDelta(44.0, info.EnergyNow, 0.01)
	assert.False(info.PluggedIn())

	capLt30 := func(i Info) bool { return i.Capacity < 30 }

	bat0 := Default().
		UrgentWhen(capLt30).
		OutputTemplate(outputs.TextTemplate(`{{.Status}}`)).
		RefreshInterval(50 * time.Millisecond)

	bat1 := New("BAT1").
		OutputTemplate(outputs.TextTemplate(`{{.RemainingPct}}`)).
		OutputColor(func(i Info) bar.Color {
			return bar.Color("#ff0000")
		})

	bat2 := New("BAT2").
		UrgentWhen(capLt30).
		OutputFunc(func(i Info) bar.Output {
			return bar.NewSegment(i.Technology)
		}).
		RefreshInterval(150 * time.Millisecond)

	m0 := testModule.NewOutputTester(t, bat0)
	m1 := testModule.NewOutputTester(t, bat1)
	m2 := testModule.NewOutputTester(t, bat2)

	out := m0.AssertOutput("on start")
	assert.Equal(
		bar.NewSegment("Charging").Urgent(false),
		out[0])

	out = m1.AssertOutput("on start")
	assert.Equal(
		bar.NewSegment("100").Color(bar.Color("#ff0000")),
		out[0])

	out = m2.AssertOutput("on start")
	assert.Equal(
		bar.NewSegment("NiCd").Urgent(false),
		out[0])

	write(battery{
		"NAME":               "BAT2",
		"STATUS":             "Discharging",
		"PRESENT":            1,
		"TECHNOLOGY":         "NiCd",
		"VOLTAGE_NOW":        20 * micros,
		"CURRENT_NOW":        int(0.5 * float64(micros)),
		"CHARGE_FULL_DESIGN": 5 * micros,
		"CHARGE_FULL":        int(4.4 * float64(micros)),
		"CHARGE_NOW":         int(1.1 * float64(micros)),
		"CAPACITY":           25,
	})

	m0.AssertOutput("on elapsed interval")

	out = m2.AssertOutput("on elapsed interval")
	assert.Equal(
		bar.NewSegment("NiCd").Urgent(true),
		out[0],
		"module picks up updates to battery info")

	// Default interval is 3 seconds,
	// but output tester only waits up to 1 second.
	m1.AssertNoOutput("when update interval has not elapsed")

	bat1.RefreshInterval(time.Hour)
	m1.AssertNoOutput("On change of refresh interval")

	bat1.OutputTemplate(outputs.TextTemplate(`{{.Capacity}}`))
	out = m1.AssertOutput("when output template changes")
	assert.Equal(
		bar.NewSegment("100").Color(bar.Color("#ff0000")),
		out[0])

	bat1.OutputColor(nil)
	out = m1.AssertOutput("when colour func changes")
	assert.Equal(bar.NewSegment("100"), out[0])

	bat1.UrgentWhen(capLt30)
	out = m1.AssertOutput("when urgent func changes")
	assert.Equal(
		bar.NewSegment("100").Urgent(false),
		out[0])
}
