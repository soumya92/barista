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

package diskspace

import (
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

type statFsResult struct {
	unix.Statfs_t
	error
}

var syncMutex sync.Mutex
var results = make(map[string]statFsResult)

func shouldError(path string, err error) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	results[path] = statFsResult{error: err}
}

func shouldReturn(path string, statfsT unix.Statfs_t) {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	results[path] = statFsResult{Statfs_t: statfsT}
}

var mockStatfs = func(path string, out *unix.Statfs_t) error {
	syncMutex.Lock()
	defer syncMutex.Unlock()
	res, ok := results[path]
	if !ok {
		return os.ErrNotExist
	}
	if res.error != nil {
		return res.error
	}
	*out = res.Statfs_t
	return nil
}

func TestDiskspace(t *testing.T) {
	assert := assert.New(t)
	statfs = mockStatfs
	scheduler.TestMode(true)

	diskspace := New("/")
	tester := testModule.NewOutputTester(t, diskspace)

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 1000,
		Bfree:  1500,
		Blocks: 2000,
	})

	out := tester.AssertOutput("on start")
	assert.Equal(outputs.Text("0.50 GB").Segments(), out)

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 200,
		Bfree:  500,
		Blocks: 2000,
	})
	tester.AssertNoOutput("until refresh")

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 0,
		Bfree:  0,
		Blocks: 2000,
	})
	scheduler.NextTick()
	out = tester.AssertOutput("on next tick")
	assert.Equal(outputs.Text("2.00 GB").Segments(), out)

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 500,
		Bfree:  800,
		Blocks: 2000,
	})
	diskspace.OutputTemplate(outputs.TextTemplate(`{{.Available.In "MB" | printf "%.1f"}}`))
	out = tester.AssertOutput("on output format change")
	assert.Equal(outputs.Text("500.0").Segments(), out)

	diskspace.UrgentWhen(func(i Info) bool {
		return i.AvailFrac() < 0.5
	})
	out = tester.AssertOutput("on urgent function change")
	assert.Equal(bar.TextSegment("500.0").Urgent(true), out[0])

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 1500,
		Bfree:  1500,
		Blocks: 2000,
	})
	diskspace.OutputColor(func(i Info) bar.Color {
		return bar.Color("red")
	})
	out = tester.AssertOutput("on color function change")
	assert.Equal(bar.TextSegment("1500.0").
		Urgent(false).
		Color(bar.Color("red")),
		out[0])

	diskspace.RefreshInterval(time.Minute)
	tester.AssertNoOutput("on refresh interval change")

	beforeTick := scheduler.Now()
	afterTick := scheduler.NextTick()
	tester.AssertOutput("on next tick")
	assert.Equal(time.Minute, afterTick.Sub(beforeTick))

	shouldError("/", os.ErrNotExist)
	scheduler.NextTick()
	tester.AssertError("on next tick after unmount")
}

func TestDiskspaceInfo(t *testing.T) {
	assert := assert.New(t)
	statfs = mockStatfs
	scheduler.TestMode(true)

	infos := make(chan Info)

	diskspace := New("/")
	diskspace.OutputFunc(func(i Info) bar.Output {
		infos <- i
		return outputs.Empty()
	})

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 800,
		Bfree:  1000,
		Blocks: 3000,
	})

	go diskspace.Stream()
	info := <-infos

	assert.InDelta(0.266, info.AvailFrac(), 0.001)
	assert.Equal(27, info.AvailPct())
	assert.Equal(67, info.UsedPct())

	assert.Equal("3.0 GB", info.Total.SI())
	assert.Equal("763 MiB", info.Available.IEC())

	assert.InDelta(2000*1000*1000, info.Used().In("foobar"), 0.001)
	assert.InDelta(0.8, info.Available.In("GB"), 0.001)

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1024 * 1024,
		Bavail: 800,
		Bfree:  1000,
		Blocks: 3000,
	})
	scheduler.NextTick()
	info = <-infos

	assert.Equal("3.1 GB", info.Total.SI())
	assert.Equal("800 MiB", info.Available.IEC())

	assert.InDelta(2000*1024*1024, info.Used().In("foobar"), 0.001)
	assert.InDelta(800.0, info.Available.In("MiB"), 0.001)
}

func TestNonexistentDiskspace(t *testing.T) {
	assert := assert.New(t)
	statfs = mockStatfs
	scheduler.TestMode(true)

	diskspace := New("/not/yet/mounted")
	tester := testModule.NewOutputTester(t, diskspace)

	tester.AssertError("on start")

	scheduler.NextTick()
	tester.AssertError("on subsequent ticks if not mounted")

	shouldReturn("/not/yet/mounted", unix.Statfs_t{
		Bsize:  1000,
		Bavail: 25 * 100 * 1000,
		Bfree:  3 * 1000 * 1000,
		Blocks: 9 * 1000 * 1000,
	})
	scheduler.NextTick()
	out := tester.AssertOutput("on next tick after mounting")
	assert.Equal(outputs.Text("6.00 GB").Segments(), out)
}
