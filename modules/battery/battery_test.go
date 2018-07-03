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
	"image/color"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
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
	batteryFile := fmt.Sprintf(
		"/sys/class/power_supply/%s/uevent",
		battery["NAME"].(string))
	afero.WriteFile(fs, batteryFile, buffer.Bytes(), 0644)
}

func TestDisconnected(t *testing.T) {
	fs = afero.NewMemMapFs()

	// No battery.
	info := batteryInfo("BAT0")
	assert.Equal(t, "Disconnected", info.Status)

	// Make sure nothing panics when values are missing.
	assert.InDelta(t, 0, info.Remaining(), 0.000001)
	assert.Equal(t, 0, info.RemainingPct())
	assert.Equal(t, time.Duration(0), info.RemainingTime())
}

func TestUnknownAndMissingStatus(t *testing.T) {
	fs = afero.NewMemMapFs()

	// No battery.
	info := batteryInfo("BAT0")
	assert.Equal(t, "Disconnected", info.Status)

	// Unknown status.
	write(battery{"NAME": "BAT1"})
	info = batteryInfo("BAT1")
	assert.Equal(t, "Unknown", info.Status)

	write(battery{
		"NAME":   "BAT2",
		"STATUS": "OtherStatus",
	})
	info = batteryInfo("BAT2")
	assert.Equal(t, "OtherStatus", info.Status)
}

func TestGarbageFiles(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()

	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/uevent",
		[]byte(`
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

	testBar.New(t)

	bat0 := Default().
		UrgentWhen(capLt30).
		OutputTemplate(`{{.Status}}`)

	bat1 := New("BAT1").
		OutputColor(func(i Info) color.Color {
			return colors.Hex("#ff0000")
		})

	bat2 := New("BAT2").
		UrgentWhen(capLt30).
		OutputFunc(func(i Info) bar.Output {
			return outputs.Text(i.Technology)
		})

	testBar.Run(bat0, bat1, bat2)

	testBar.LatestOutput().AssertEqual(outputs.Group(
		outputs.Text("Charging").Urgent(false),
		outputs.Text("BATT 100%").Color(colors.Hex("#ff0000")),
		outputs.Text("NiCd").Urgent(false),
	), "on start")

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
	testBar.Tick()

	testBar.LatestOutput().At(2).AssertEqual(
		bar.TextSegment("NiCd").Urgent(true),
		"module picks up updates to battery info")

	bat1.RefreshInterval(time.Hour)
	testBar.AssertNoOutput("On change of refresh interval")

	bat1.OutputTemplate(`{{.Capacity}}`)
	testBar.NextOutput().At(1).AssertEqual(
		bar.TextSegment("100").Color(colors.Hex("#ff0000")),
		"when output template changes")

	bat1.OutputColor(nil)
	testBar.NextOutput().At(1).AssertEqual(
		bar.TextSegment("100"), "when colour func changes")

	bat1.UrgentWhen(capLt30)
	testBar.NextOutput().At(1).AssertEqual(
		bar.TextSegment("100").Urgent(false),
		"when urgent func changes")
}
