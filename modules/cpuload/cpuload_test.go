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
	"sync"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	"barista.run/timing"

	"github.com/stretchr/testify/require"
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
	require := require.New(t)
	getloadavg = mockloadavg
	testBar.New(t)

	shouldReturn(0, 0, 0)

	load := New()
	testBar.Run(load)

	testBar.NextOutput().AssertText(
		[]string{"0.00"}, "on start")

	shouldReturn(1, 2, 3)
	testBar.AssertNoOutput("until refresh")
	testBar.Tick()
	testBar.NextOutput().AssertText(
		[]string{"1.00"}, "on next tick")

	load.Output(func(l LoadAvg) bar.Output {
		return outputs.Textf("%.2f", l.Min5()).Urgent(l.Min15() > 2)
	})
	testBar.NextOutput().AssertEqual(
		outputs.Text("2.00").Urgent(true),
		"on output format change")

	shouldReturn(0, 0, 0)
	testBar.Tick()
	testBar.NextOutput().AssertEqual(
		outputs.Text("0.00").Urgent(false),
		"on next tick")

	load.RefreshInterval(time.Minute)
	testBar.AssertNoOutput("on refresh interval change")

	beforeTick := timing.Now()
	afterTick := timing.NextTick()
	testBar.NextOutput().Expect("on next tick")
	require.Equal(time.Minute, afterTick.Sub(beforeTick))

	testBar.AssertNoOutput("until next tick")
}

func TestErrors(t *testing.T) {
	require := require.New(t)
	getloadavg = mockloadavg
	testBar.New(t)

	shouldReturn(1)
	load := New()
	testBar.Run(load)

	errs := testBar.NextOutput().AssertError("on start with error")
	require.Equal("getloadavg: 1", errs[0], "error string contains getloadavg code")

	shouldReturn(1, 2, 3, 4, 5)
	out := testBar.NextOutput("with restart click handler")
	out.At(0).LeftClick()
	testBar.NextOutput().Expect("on restart, clears error segment")
	errs = testBar.NextOutput().AssertError("on restart with error")
	require.Equal("getloadavg: 5", errs[0], "error string contains getloadavg code")

	shouldError(errors.New("test"))
	out = testBar.NextOutput("with restart click handler")
	out.At(0).LeftClick()
	testBar.NextOutput().Expect("on restart, clears error segment")
	errs = testBar.NextOutput().AssertError("on restart with error")
	require.Equal("test", errs[0], "error string is passed through")
}
