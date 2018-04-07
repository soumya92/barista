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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
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
	scheduler.TestMode(true)

	load := New()
	tester := testModule.NewOutputTester(t, load)

	shouldReturn(0, 0, 0)

	out := tester.AssertOutput("on start")
	assert.Equal(bar.NewSegment("0.00"), out[0])

	shouldReturn(1, 2, 3)
	tester.AssertNoOutput("until refresh")

	scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(bar.NewSegment("1.00"), out[0])

	load.OutputTemplate(outputs.TextTemplate(`{{.Min5 | printf "%.2f"}}`))
	out = tester.AssertOutput("on output format change")
	assert.Equal(bar.NewSegment("2.00"), out[0])

	load.UrgentWhen(func(l LoadAvg) bool {
		return l.Min15() > 2
	})
	out = tester.AssertOutput("on urgent function change")
	assert.Equal(bar.NewSegment("2.00").Urgent(true), out[0])

	load.OutputColor(func(l LoadAvg) bar.Color {
		return bar.Color("red")
	})
	out = tester.AssertOutput("on color function change")
	assert.Equal(bar.NewSegment("2.00").
		Urgent(true).
		Color(bar.Color("red")),
		out[0])

	shouldReturn(0, 0, 0)
	scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(bar.NewSegment("0.00").
		Urgent(false).
		Color(bar.Color("red")),
		out[0])

	shouldReturn(1)
	scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(outputs.Error(fmt.Errorf("getloadavg: 1")).Segments(), out)

	shouldReturn(1, 2, 3, 4, 5)
	scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(outputs.Error(fmt.Errorf("getloadavg: 5")).Segments(), out)

	shouldError(fmt.Errorf("test"))
	scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(outputs.Error(fmt.Errorf("test")).Segments(), out)

	load.RefreshInterval(time.Minute)
	tester.AssertNoOutput("on refresh interval change")

	beforeTick := scheduler.Now()
	afterTick := scheduler.NextTick()
	tester.AssertOutput("on next tick")
	assert.Equal(time.Minute, afterTick.Sub(beforeTick))
}
