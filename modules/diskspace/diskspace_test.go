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
	"image/color"
	"os"
	"sync"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
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
	require := require.New(t)
	statfs = mockStatfs
	testBar.New(t)

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 1000,
		Bfree:  1500,
		Blocks: 2000,
	})

	diskspace := New("/")
	testBar.Run(diskspace)
	testBar.LatestOutput().AssertText([]string{"0.50 GB"}, "on start")

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 200,
		Bfree:  500,
		Blocks: 2000,
	})
	testBar.AssertNoOutput("until refresh")

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 0,
		Bfree:  0,
		Blocks: 2000,
	})
	testBar.Tick()
	testBar.NextOutput().AssertText([]string{"2.00 GB"}, "on next tick")

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 500,
		Bfree:  800,
		Blocks: 2000,
	})
	diskspace.Template(`{{.Available.Megabytes | printf "%.1f"}}`)
	testBar.NextOutput().AssertText(
		[]string{"0.0"}, "on output format change, updates with existing data")

	diskspace.UrgentWhen(func(i Info) bool {
		return i.AvailFrac() < 0.5
	})
	testBar.NextOutput().AssertEqual(
		outputs.Text("0.0").Urgent(true),
		"on urgent function change, updates with existing data")

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 1500,
		Bfree:  1500,
		Blocks: 2000,
	})
	testBar.Tick()
	testBar.NextOutput().Expect("on tick")

	diskspace.OutputColor(func(i Info) color.Color {
		return colors.Hex("#f00")
	})
	testBar.NextOutput().AssertEqual(
		outputs.Text("1500.0").Urgent(false).Color(colors.Hex("#f00")),
		"on color function change")

	diskspace.RefreshInterval(time.Minute)
	testBar.AssertNoOutput("on refresh interval change")

	beforeTick := timing.Now()
	afterTick := timing.NextTick()
	testBar.NextOutput().Expect("on next tick")
	require.Equal(time.Minute, afterTick.Sub(beforeTick))

	shouldError("/", os.ErrPermission)
	testBar.Tick()
	testBar.NextOutput().AssertError("on tick with error")
	testBar.Tick()
	testBar.AssertNoOutput("on subsequent tick with error")
}

func TestDiskspaceInfo(t *testing.T) {
	require := require.New(t)
	statfs = mockStatfs
	testBar.New(t)

	infos := make(chan Info)

	diskspace := New("/")
	diskspace.Output(func(i Info) bar.Output {
		infos <- i
		return nil
	})

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1000 * 1000,
		Bavail: 800,
		Bfree:  1000,
		Blocks: 3000,
	})

	testBar.Run(diskspace)
	info := <-infos

	// Setting the output function sometimes causes an update,
	// so read the latest info if available.
	select {
	case info = <-infos:
	default:
	}

	require.InDelta(0.266, info.AvailFrac(), 0.001)
	require.Equal(27, info.AvailPct())
	require.Equal(67, info.UsedPct())

	require.InDelta(3.0, info.Total.Gigabytes(), float64(unit.Byte))
	require.InDelta(763, info.Available.Mebibytes(), float64(unit.Byte))

	shouldReturn("/", unix.Statfs_t{
		Bsize:  1024 * 1024,
		Bavail: 800,
		Bfree:  1000,
		Blocks: 3000,
	})
	testBar.Tick()
	info = <-infos

	require.InDelta(3.1, info.Total.Gigabytes(), float64(unit.Byte))
	require.InDelta(800, info.Available.Mebibytes(), float64(unit.Byte))
}

func TestNonexistentDiskspace(t *testing.T) {
	statfs = mockStatfs
	testBar.New(t)

	diskspace := New("/not/yet/mounted")
	testBar.Run(diskspace)

	testBar.NextOutput().AssertEmpty(
		"unmounted disk shows no output")
	testBar.Tick()
	testBar.NextOutput().AssertEmpty(
		"on tick when not mounted")

	shouldReturn("/not/yet/mounted", unix.Statfs_t{
		Bsize:  1000,
		Bavail: 25 * 100 * 1000,
		Bfree:  3 * 1000 * 1000,
		Blocks: 9 * 1000 * 1000,
	})
	testBar.Tick()
	testBar.NextOutput().AssertEqual(
		outputs.Text("6.00 GB"), "on next tick after mounting")
}
