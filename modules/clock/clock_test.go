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
	"github.com/soumya92/barista/base/scheduler"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestSimple(t *testing.T) {
	assert := assert.New(t)
	scheduler.TestMode(true)
	fixedTime := time.Date(2017, time.March, 1, 0, 0, 0, 0, time.Local)
	scheduler.AdvanceTo(fixedTime)

	local := New()
	tester := testModule.NewOutputTester(t, local)

	out := tester.AssertOutput("on start")
	assert.Equal(bar.NewSegment("00:00"), out[0])

	now := scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(bar.NewSegment("00:00"), out[0])
	assert.Equal(1, now.Second(), "increases by granularity")
	assert.Equal(0, now.Nanosecond(), "triggers at exact granularity")

	scheduler.AdvanceBy(500 * time.Millisecond)
	tester.AssertNoOutput("less than granularity")

	now = scheduler.NextTick()
	assert.Equal(2, now.Second(), "increases by granularity")
	assert.Equal(0, now.Nanosecond(), "triggers at exact granularity")
	tester.AssertOutput("on next tick")

	local.Granularity(time.Minute)
	tester.AssertOutput("on granularity change")

	local.OutputFormat("15:04:05")
	out = tester.AssertOutput("on output format change")
	assert.Equal(bar.NewSegment("00:00:02"), out[0])

	now = scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(bar.NewSegment("00:01:00"), out[0])
	assert.Equal(0, now.Second(), "triggers on exact granularity")
	assert.Equal(1, now.Minute(), "triggers on exact granularity")

	tester.AssertNoOutput("when time is frozen")
	// This also serves as a check to make sure we've consumed all outputs.
}

func TestZones(t *testing.T) {
	assert := assert.New(t)
	scheduler.TestMode(true)
	fixedTime := time.Date(2017, time.March, 1, 13, 15, 0, 0, time.UTC)
	scheduler.AdvanceTo(fixedTime)

	pst := New().Timezone("America/Los_Angeles").OutputFormat("15:04:05")
	tPst := testModule.NewOutputTester(t, pst)

	berlin := New().Timezone("Europe/Berlin").OutputFormat("15:04:05")
	tBerlin := testModule.NewOutputTester(t, berlin)

	tokyo := New().Timezone("Asia/Tokyo").OutputFormat("15:04:05")
	tTokyo := testModule.NewOutputTester(t, tokyo)

	unknown := New().Timezone("Global/Unknown").OutputFormat("15:04:05")
	tUnknown := testModule.NewOutputTester(t, unknown)

	out := tPst.AssertOutput("on start")
	assert.Equal(bar.NewSegment("05:15:00"), out[0])

	out = tBerlin.AssertOutput("on start")
	assert.Equal(bar.NewSegment("14:15:00"), out[0])

	out = tTokyo.AssertOutput("on start")
	assert.Equal(bar.NewSegment("22:15:00"), out[0])

	out = tUnknown.AssertOutput("on start with error")
	assert.Contains(out[0].Text(), "Global/Unknown")
	assert.True(out[0]["urgent"].(bool))

	scheduler.NextTick()

	tPst.AssertOutput("on tick")
	tBerlin.AssertOutput("on tick")
	tTokyo.AssertOutput("on tick")
	tUnknown.AssertNoOutput("on tick with error")
}
