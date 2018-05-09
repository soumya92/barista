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

func TestSimple(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(fixedTime)

	testBar.Run(New().Granularity(time.Minute))
	testBar.LatestOutput().AssertText(
		[]string{"00:00"}, "on start")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:01"}, "on next tick")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"00:02"}, "on next tick")
}

func TestGranularity(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(fixedTime)

	local := New().OutputFunc(func(now time.Time) bar.Output {
		return outputs.Text(now.Format("15:04:05"))
	}).Granularity(time.Hour)
	testBar.Run(local)
	testBar.LatestOutput().AssertText(
		[]string{"00:00:00"}, "on start")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"01:00:00"}, "on tick")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"02:00:00"}, "on tick")

	local.Granularity(time.Minute)
	testBar.NextOutput().AssertText(
		[]string{"02:00:00"}, "on granularity change")

	scheduler.NextTick()
	testBar.NextOutput().AssertText(
		[]string{"02:01:00"}, "on tick")
}

func TestZones(t *testing.T) {
	testBar.New(t)
	scheduler.AdvanceTo(
		time.Date(2017, time.March, 1, 13, 15, 0, 0, time.UTC))

	pst := New().Timezone("America/Los_Angeles").OutputFormat("15:04:05")
	berlin := New().Timezone("Europe/Berlin").OutputFormat("15:04:05")
	tokyo := New().Timezone("Asia/Tokyo").OutputFormat("15:04:05")
	unknown := New().Timezone("Global/Unknown").OutputFormat("15:04:05")

	testBar.Run(pst, berlin, tokyo, unknown)

	out := testBar.LatestOutput()
	out.At(0).AssertText("05:15:00", "on start")
	out.At(1).AssertText("14:15:00")
	out.At(2).AssertText("22:15:00")
	errStr := out.At(3).AssertError("Invalid timezone")
	assert.Contains(t, errStr, "Global/Unknown", "error mentions time zone")

	scheduler.NextTick()
	testBar.LatestOutput().Expect("on tick")
}
