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
	"strings"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
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
	batteryTypeFile := fmt.Sprintf(
		"/sys/class/power_supply/%s/type",
		battery["NAME"].(string))
	if strings.HasPrefix(battery["NAME"].(string), "BAT") {
		afero.WriteFile(fs, batteryTypeFile, []byte("Battery\n"), 0644)
	} else {
		afero.WriteFile(fs, batteryTypeFile, []byte("Mains\n"), 0644)
	}
}

func TestDisconnected(t *testing.T) {
	fs = afero.NewMemMapFs()
	require := require.New(t)

	// No battery.
	info := batteryInfo("BAT0")
	require.Equal(Disconnected, info.Status)

	// Make sure nothing panics when values are missing.
	require.InDelta(0, info.Remaining(), 0.000001)
	require.Equal(0, info.RemainingPct())
	require.Equal(time.Duration(0), info.RemainingTime())
}

func TestUnknownAndMissingStatus(t *testing.T) {
	fs = afero.NewMemMapFs()
	require := require.New(t)

	// No battery.
	info := batteryInfo("BAT0")
	require.Equal(Disconnected, info.Status)

	info = allBatteriesInfo()
	require.Equal(Disconnected, info.Status)

	// Unknown status.
	write(battery{"NAME": "BAT1"})
	info = batteryInfo("BAT1")
	require.Equal(Unknown, info.Status)

	info = allBatteriesInfo()
	require.Equal(Unknown, info.Status)

	write(battery{
		"NAME":   "BAT2",
		"STATUS": "OtherStatus",
	})
	info = batteryInfo("BAT2")
	require.Equal(Unknown, info.Status)

	info = allBatteriesInfo()
	require.Equal(Unknown, info.Status)

	require.False(info.Discharging(), "Unknown battery is not discharging")
}

func TestGarbageFiles(t *testing.T) {
	require := require.New(t)
	fs = afero.NewMemMapFs()

	afero.WriteFile(fs, "/sys/class/power_supply/BAT0/uevent",
		[]byte(`
POWER_SUPPLY_NAME=BAT0
POWER_SUPPLY_STATUS=Disconnected
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

	require.Equal(Disconnected, info.Status)
	require.InDelta(12, info.Voltage, 0.01)
	require.InDelta(6, info.Power, 0.01)
	require.InDelta(60, info.EnergyFull, 0.01)
	require.InDelta(36, info.EnergyNow, 0.01)
	// invalid entry is not parsed.
	require.Equal(0.0, info.EnergyMax)
	// invalid entry does not overwrite previous.
	require.Equal("NiCd", info.Technology)

	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/sys/class/power_supply", []byte(`foobar`), 0644)
	info = allBatteriesInfo()
	require.Equal(Unknown, info.Status)
}

var micros = 1000 * 1000

func TestSimple(t *testing.T) {
	require := require.New(t)
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

	// (Pinebook Pro) example without CHARGE_NOW
	write(battery{
		"NAME":               "BAT3",
		"STATUS":             "Discharging",
		"PRESENT":            1,
		"TECHNOLOGY":         "Li-ion",
		"VOLTAGE_NOW":        20 * micros,
		"CURRENT_NOW":        int(0.5 * float64(micros)),
		"CHARGE_FULL_DESIGN": 5 * micros,
		"CHARGE_FULL":        int(4.4 * float64(micros)),
		"CAPACITY":           50,
	})

	info := batteryInfo("BAT0")
	require.Equal(Charging, info.Status)
	require.Equal("Li-poly", info.Technology)
	require.InDelta(20.0, info.Voltage, 0.01)
	require.InDelta(10.0, info.Power, 0.01)
	require.InDelta(0.5, info.Remaining(), 0.0001)
	require.InDelta(20.0, info.EnergyNow, 0.01)
	require.InDelta(40.0, info.EnergyFull, 0.01)
	require.InDelta(50.0, info.EnergyMax, 0.01)
	require.Equal(2*time.Hour, info.RemainingTime())
	require.Equal(50, info.Capacity)
	require.True(info.PluggedIn())

	info = batteryInfo("BAT1")
	require.Equal(Full, info.Status)
	require.True(info.PluggedIn())

	info = batteryInfo("BAT2")
	require.InDelta(20.0, info.Voltage, 0.01)
	require.InDelta(100.0, info.EnergyMax, 0.01)
	require.InDelta(88.0, info.EnergyFull, 0.01)
	require.InDelta(44.0, info.EnergyNow, 0.01)
	require.False(info.PluggedIn())

	info = batteryInfo("BAT3")
	require.InDelta(20.0, info.Voltage, 0.01)
	require.InDelta(100.0, info.EnergyMax, 0.01)
	require.InDelta(88.0, info.EnergyFull, 0.01)
	require.InDelta(44.0, info.EnergyNow, 0.01)
	require.False(info.PluggedIn())

	testBar.New(t)

	bat0 := Named("BAT0").Output(func(i Info) bar.Output {
		return outputs.Textf("%s", i.Status)
	})
	bat1 := Named("BAT1")
	bat2 := Named("BAT2").Output(func(i Info) bar.Output {
		return outputs.Text(i.Technology).Urgent(i.Capacity < 30)
	})

	testBar.Run(bat0, bat1, bat2)

	testBar.LatestOutput().AssertEqual(outputs.Group(
		outputs.Text("Charging"),
		outputs.Text("BATT 100%"),
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

	bat1.Output(func(i Info) bar.Output {
		return outputs.Textf("%d", i.Capacity)
	})
	testBar.NextOutput().At(1).AssertEqual(
		bar.TextSegment("100"),
		"when output format changes")
}

func TestCombined(t *testing.T) {
	require := require.New(t)
	fs = afero.NewMemMapFs()

	bat0 := battery{
		"NAME":               "BAT0",
		"STATUS":             "Charging",
		"PRESENT":            1,
		"TECHNOLOGY":         "Li-poly",
		"VOLTAGE_NOW":        20 * micros,
		"POWER_NOW":          10 * micros,
		"ENERGY_FULL_DESIGN": 50 * micros,
		"ENERGY_FULL":        50 * micros,
		"ENERGY_NOW":         25 * micros,
		"CAPACITY":           50,
	}
	bat1 := battery{
		"NAME":               "BAT1",
		"STATUS":             "Not charging",
		"PRESENT":            1,
		"VOLTAGE_NOW":        10 * micros,
		"POWER_NOW":          0 * micros,
		"ENERGY_FULL_DESIGN": 50 * micros,
		"ENERGY_FULL":        50 * micros,
		"ENERGY_NOW":         20 * micros,
		"CAPACITY":           100,
	}
	bat2 := battery{
		"NAME":               "BAT2",
		"STATUS":             "Discharging",
		"PRESENT":            1,
		"TECHNOLOGY":         "NiCd",
		"VOLTAGE_NOW":        5 * micros,
		"CURRENT_NOW":        1 * micros,
		"CHARGE_FULL_DESIGN": 10 * micros,
		"CHARGE_FULL":        10 * micros,
		"CHARGE_NOW":         5 * micros,
		"CAPACITY":           50,
	}
	ac := battery{
		"NAME":   "AC",
		"ONLINE": 0,
	}

	write(ac)
	info := allBatteriesInfo()
	require.Equal(Disconnected, info.Status)

	writeAll := func() { write(bat0); write(bat1); write(bat2); write(ac) }
	writeAll()

	testBar.New(t)
	testBar.Run(All().Output(func(i Info) bar.Output {
		return outputs.Textf("%s - %v/%v", i.Status, i.RemainingPct(), i.RemainingTime())
	}))

	info = allBatteriesInfo()
	require.Equal("Li-poly,NiCd", info.Technology)
	require.InDelta(11.7857142, info.Voltage, 1.0/float64(micros))

	// Total capacity: 150Wh, currently available: 25Wh + 20Wh + 25Wh = 70Wh.
	// Net to be charged: 80Wh, net charge rate: 10W - 5W =  5W.
	testBar.NextOutput().AssertText([]string{
		"Charging - 46/16h0m0s"}, "on start")

	bat0["STATUS"] = "Discharging"
	bat2["CURRENT_NOW"] = 2 * micros
	writeAll()
	testBar.Tick()

	// Available: 70Wh, net discharge rate: 10W + 10W = 20W.
	testBar.NextOutput().AssertText([]string{
		"Discharging - 46/3h30m0s"})

	bat0["ENERGY_NOW"] = 30 * micros
	bat0["CAPACITY"] = 60
	bat2["STATUS"] = "Charging"
	bat2["CURRENT_NOW"] = 5 * micros
	writeAll()
	testBar.Tick()

	// Available: 75Wh, Total: 150Wh.
	// To charge: 75Wh, net charge rate: 25W - 10W = 15W.
	testBar.NextOutput().AssertText([]string{
		"Charging - 50/5h0m0s"})

	bat0["STATUS"] = "Charging"
	bat2["STATUS"] = "Discharging"
	writeAll()
	testBar.Tick()

	// Available: 75Wh, Total: 150Wh.
	// Net discharge rate: 25W - 10W = 15W.
	testBar.NextOutput().AssertText([]string{
		"Discharging - 50/5h0m0s"})
}
