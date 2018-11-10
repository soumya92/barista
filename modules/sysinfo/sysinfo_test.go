// Copyright 2018 Google Inc.
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

package sysinfo

import (
	"errors"
	"sync"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/format"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	"barista.run/timing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var syncMutex sync.Mutex
var simulatedInfo unix.Sysinfo_t
var simulatedErr error

func shouldError(err error) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	simulatedErr = err
	simulatedInfo = unix.Sysinfo_t{}
}

func shouldReturn(info unix.Sysinfo_t) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	simulatedErr = nil
	simulatedInfo = info
}

var mockSysinfo = func(out *unix.Sysinfo_t) error {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	*out = simulatedInfo
	return simulatedErr
}

func resetForTest() {
	shouldReturn(unix.Sysinfo_t{})
	sysinfo = mockSysinfo
	currentInfo = &value.ErrorValue{}
	once = sync.Once{}
	construct()
	// Flush upates for test.
	n := currentInfo.Next()
	update()
	<-n
}

func TestSysinfo(t *testing.T) {
	require := require.New(t)
	testBar.New(t)
	resetForTest()

	load := New()
	uptime := New()
	procs := New()
	swap := New()
	testBar.Run(load, uptime, procs, swap)
	testBar.LatestOutput().Expect("on start")

	load.Output(func(s Info) bar.Output {
		return outputs.Textf("%v", s.Loads[0])
	})
	uptime.Output(func(s Info) bar.Output {
		return outputs.Textf("%v", s.Uptime)
	})
	procs.Output(func(s Info) bar.Output {
		return outputs.Textf("%d", s.Procs)
	})
	swap.Output(func(s Info) bar.Output {
		return outputs.Text(format.IBytesize(s.TotalSwap))
	})
	testBar.LatestOutput().AssertText(
		[]string{"0", "0s", "0", "0 B"}, "on template change")

	shouldReturn(unix.Sysinfo_t{
		Procs:     4,
		Unit:      1024 * 1024,
		Totalswap: 512,
		Uptime:    3600,
		Loads:     [3]uint64{65536, 32767, 0},
	})
	testBar.AssertNoOutput("until refresh")
	testBar.Tick()
	testBar.LatestOutput().AssertText(
		[]string{"1", "1h0m0s", "4", "512 MiB"}, "on next tick")

	load.Output(func(s Info) bar.Output {
		return outputs.Textf("%.2f", s.Loads[1])
	})
	testBar.LatestOutput(0).At(0).
		AssertText("0.50", "on output format change")

	RefreshInterval(time.Minute)
	testBar.AssertNoOutput("on refresh interval change")

	beforeTick := timing.Now()
	afterTick := timing.NextTick()
	testBar.LatestOutput().Expect("on next tick")
	require.Equal(time.Minute, afterTick.Sub(beforeTick))

	testBar.AssertNoOutput("until next tick")
}

func TestErrors(t *testing.T) {
	require := require.New(t)
	testBar.New(t)
	resetForTest()

	testBar.Run(New())
	testBar.NextOutput().Expect("on start")

	shouldError(errors.New("test"))
	testBar.Tick()

	errs := testBar.NextOutput().AssertError("on next tick with error")
	require.Equal("test", errs[0], "error string is passed through")

	shouldError(errors.New("something else"))
	testBar.Tick()
	errs = testBar.NextOutput().AssertError("on next tick with error")
	require.Equal("something else", errs[0], "new error is propagated")
}
