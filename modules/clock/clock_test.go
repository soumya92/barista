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

package clock

import (
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/scheduler"
	testBar "github.com/soumya92/barista/testing/bar"
)

var fixedTime = time.Date(2017, time.March, 1, 0, 0, 0, 0, time.Local)

func TestSimpleTicking(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(fixedTime)

	testBar.Run(Local())
	testBar.LatestOutput().AssertText(
		[]string{"00:00"}, "on start")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:01"}, "on next tick")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:02"}, "on next tick")
}

func TestAutoGranularities(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(fixedTime)
	assert := assert.New(t)

	local := Local().OutputFormat("15:04:05")
	testBar.Run(local)
	testBar.LatestOutput().AssertText(
		[]string{"00:00:00"}, "on start")

	now := scheduler.NextTick()
	testBar.LatestOutput().AssertText(
		[]string{"00:00:01"}, "on next tick")
	assert.Equal(1, now.Second(), "increases by granularity")
	assert.Equal(0, now.Nanosecond(), "triggers at exact granularity")

	scheduler.AdvanceBy(500 * time.Millisecond)
	testBar.AssertNoOutput("less than granularity")

	now = scheduler.NextTick()
	assert.Equal(2, now.Second(), "increases by granularity")
	assert.Equal(0, now.Nanosecond(), "triggers at exact granularity")
	testBar.NextOutput().AssertText(
		[]string{"00:00:02"}, "on next tick")

	local.OutputFormat("15:04")
	testBar.NextOutput().AssertText(
		[]string{"00:00"}, "on output format change")

	now = scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:01"}, "on next tick")
	assert.Equal(1, now.Minute(), "triggers on exact granularity")
	assert.Equal(0, now.Second(), "triggers on exact granularity")

	local.OutputFormat("15:04:05.0")
	testBar.NextOutput().AssertText(
		[]string{"00:01:00.0"}, "on output format change")
	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:01:00.1"}, "on next tick")

	local.OutputFormat("15:04:05.000")
	testBar.NextOutput().AssertText(
		[]string{"00:01:00.100"}, "on output format change")
	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:01:00.101"}, "on next tick")

	local.OutputFormat("15:04:05.00")
	testBar.NextOutput().AssertText(
		[]string{"00:01:00.10"}, "on output format change")
	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:01:00.11"}, "on next tick")

	testBar.AssertNoOutput("when time is frozen")
}

func TestManualGranularities(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(fixedTime)

	local := Local().OutputFunc(time.Hour, func(now time.Time) bar.Output {
		return outputs.Text(now.Format("15:04:05"))
	})
	testBar.Run(local)
	testBar.LatestOutput().AssertText(
		[]string{"00:00:00"}, "on start")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"01:00:00"}, "on tick")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"02:00:00"}, "on tick")

	local.OutputFunc(time.Minute, func(now time.Time) bar.Output {
		return outputs.Text(now.Format("15:04:05.00"))
	})
	testBar.NextOutput().AssertText(
		[]string{"02:00:00.00"}, "on format function + granularity change")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"02:01:00.00"}, "on tick")
}

func TestZones(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(
		time.Date(2017, time.March, 1, 13, 15, 0, 0, time.UTC))

	la, _ := time.LoadLocation("America/Los_Angeles")
	pst := Zone(la).OutputFormat("15:04:05")

	berlin, err := ZoneByName("Europe/Berlin")
	assert.NoError(t, err)
	berlin.OutputFormat("15:04:05")

	tokyo, err := ZoneByName("Asia/Tokyo")
	assert.NoError(t, err)
	tokyo.OutputFormat("15:04:05")

	testBar.Run(pst, berlin, tokyo)

	_, err = ZoneByName("Global/Unknown")
	assert.Error(t, err, "when loading unknown zone")

	testBar.LatestOutput().AssertText(
		[]string{"05:15:00", "14:15:00", "22:15:00"},
		"on start")

	scheduler.NextTick()
	testBar.LatestOutput().AssertText(
		[]string{"05:15:01", "14:15:01", "22:15:01"},
		"on tick")

	berlin.Timezone(la)
	testBar.LatestOutput().At(1).AssertText(
		"05:15:01", "on timezone change")
}
