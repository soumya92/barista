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

package cpuload

import (
	"errors"
	"image/color"
	"sync"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
)

var syncMutex sync.Mutex
var simulatedLoads LoadAvg
var simulatedCount int
var simulatedErr error

func shouldError(err error) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	simulatedErr = err
	simulatedCount = 0
	simulatedLoads = LoadAvg{}
}

func shouldReturn(loads ...float64) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	simulatedErr = nil
	simulatedCount = len(loads)
	for i, l := range loads {
		if i < len(simulatedLoads) {
			simulatedLoads[i] = l
		}
	}
}

var mockloadavg = func(out *LoadAvg, count int) (int, error) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	for i := 0; i < count; i++ {
		if i < len(simulatedLoads) {
			out[i] = simulatedLoads[i]
		}
	}
	return simulatedCount, simulatedErr
}

func TestCpuload(t *testing.T) {
	assert := assert.New(t)
	getloadavg = mockloadavg
	testBar.New(t)

	shouldReturn(0, 0, 0)

	load := New()
	testBar.Run(load)

	testBar.LatestOutput().AssertText(
		[]string{"0.00"}, "on start")

	shouldReturn(1, 2, 3)
	testBar.AssertNoOutput("until refresh")
	testBar.Tick()
	testBar.NextOutput().AssertText(
		[]string{"1.00"}, "on next tick")

	load.Template(`{{.Min5 | printf "%.2f"}}`)
	testBar.NextOutput().AssertText(
		[]string{"2.00"}, "on output format change")

	load.UrgentWhen(func(l LoadAvg) bool {
		return l.Min15() > 2
	})
	testBar.NextOutput().AssertEqual(
		outputs.Text("2.00").Urgent(true),
		"on urgent function change")

	load.OutputColor(func(l LoadAvg) color.Color {
		return colors.Hex("#f00")
	})
	testBar.NextOutput().AssertEqual(
		outputs.Text("2.00").Urgent(true).Color(colors.Hex("#f00")),
		"on color function change")

	shouldReturn(0, 0, 0)
	testBar.Tick()
	testBar.NextOutput().AssertEqual(
		outputs.Text("0.00").Urgent(false).Color(colors.Hex("#f00")),
		"on next tick")

	load.RefreshInterval(time.Minute)
	testBar.AssertNoOutput("on refresh interval change")

	beforeTick := timing.Now()
	afterTick := timing.NextTick()
	testBar.NextOutput().Expect("on next tick")
	assert.Equal(time.Minute, afterTick.Sub(beforeTick))

	testBar.AssertNoOutput("until next tick")
}

func TestErrors(t *testing.T) {
	assert := assert.New(t)
	getloadavg = mockloadavg
	testBar.New(t)

	load := New()
	testBar.Run(load)

	shouldReturn(1)
	testBar.Tick()
	errs := testBar.LatestOutput().AssertError("on next tick with error")
	assert.Equal("getloadavg: 1", errs[0], "error string contains getloadavg code")

	shouldReturn(1, 2, 3, 4, 5)
	testBar.Click(0) // to restart.
	errs = testBar.LatestOutput().AssertError("on next tick with error")
	assert.Equal("getloadavg: 5", errs[0], "error string contains getloadavg code")

	shouldError(errors.New("test"))
	testBar.Click(0)
	errs = testBar.LatestOutput().AssertError("on next tick with error")
	assert.Equal("test", errs[0], "error string is passed through")
}
