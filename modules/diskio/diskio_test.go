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

package diskio

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

type diskstats map[string][]int

func shouldReturn(stats diskstats) {
	var out bytes.Buffer
	idx := 0
	for disk, stats := range stats {
		out.WriteString(fmt.Sprintf(
			"0 %d %s all other %d fields are ignored %d * * * *\n",
			idx, disk, stats[0], stats[1]))
		idx++
	}
	lock.Lock()
	afero.WriteFile(fs, "/proc/diskstats", out.Bytes(), 0644)
	lock.Unlock()
}

// resetForTest resets diskio's shared state for testing purposes.
func resetForTest() {
	fs = afero.NewMemMapFs()
	modules = nil
	updater = nil
	once = sync.Once{}
}

func TestDiskIo(t *testing.T) {
	resetForTest()
	testBar.New(t)

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{0, 0},
	})
	construct()

	sda1 := New("sda1").Output(func(i IO) bar.Output {
		return outputs.Textf("sda1: %s", outputs.Byterate(i.Total()))
	})
	sdb1 := New("sdb1").Output(func(i IO) bar.Output {
		return outputs.Textf("sdb1: %s", outputs.IByterate(i.Total()))
	})
	testBar.Run(sda1, sdb1)

	testBar.LatestOutput(0).Expect("on start")

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{9, 9},
	})
	testBar.Tick()

	// 9+9 sectors / 3 seconds = 6 sectors / second * 512 bytes / sector = 3072 bytes.
	testBar.LatestOutput(0).At(0).AssertText("sda1: 3.1 kB/s", "on tick")

	// Simpler math.
	RefreshInterval(time.Second)

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{9, 10},
	})
	testBar.Tick()

	// only one output because other disk is missing.
	testBar.LatestOutput().AssertText(
		[]string{"sda1: 512 B/s"}, "on tick")

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{9, 20},
	})
	testBar.Tick()

	testBar.LatestOutput().AssertText(
		[]string{"sda1: 5.1 kB/s"}, "on tick")

	sda1.Output(func(i IO) bar.Output {
		return outputs.Textf("sda1: %.1f", i.Total().KibibytesPerSecond())
	})
	testBar.LatestOutput(0).AssertText(
		[]string{"sda1: 5.0"}, "on output function change")

	shouldReturn(diskstats{
		"sdb":  []int{0, 0},
		"sdb1": []int{300, 0},
	})
	testBar.Tick()

	testBar.LatestOutput().AssertEmpty(
		"first tick after disk is added/removed")

	shouldReturn(diskstats{
		"sdb":  []int{0, 0},
		"sdb1": []int{300, 100},
		"sdc":  []int{0, 0},
	})
	testBar.Tick()

	testBar.LatestOutput().AssertText(
		[]string{"sdb1: 50 KiB/s"}, "on next tick")
}

func TestErrors(t *testing.T) {
	resetForTest()
	testBar.New(t)
	construct()

	sda := New("sda")
	sda1 := New("sda1")
	sda2 := New("sda2")

	testBar.Run(sda, sda1, sda2)
	testBar.AssertNoOutput("on start with missing diskstats")

	testBar.Tick()
	testBar.LatestOutput().AssertError("on first tick if missing diskstats")

	lock.Lock()
	afero.WriteFile(fs, "/proc/diskstats", []byte(`
-- Lines in weird formats --
Empty:

Valid line:
0 0 sda 0 0 100 0 0 0 100 0 0 0 0
Too few fields:
0 0 0 0
Fields are non-numeric:
a b c d e f g h i j k l m n
a b sda1 0 0 a 0 0 0 100 0 0 0 0
a b sda2 0 0 100 0 0 0 b 0 0 0 0
`), 0644)
	lock.Unlock()
	testBar.Tick()
	out := testBar.LatestOutput()
	out.At(0).AssertError("invalid read count")
	out.At(1).AssertError("invalid write count")
	// First tick initialises the stats,
	// but we won't have a delta until the next tick.
	require.Equal(t, 2, out.Len(), "on first tick")

	lock.Lock()
	afero.WriteFile(fs, "/proc/diskstats", []byte(`
	0 0 sda 0 0 400 0 0 0 400 0 0 0 0
	`), 0644)
	lock.Unlock()
	testBar.Tick()

	testBar.LatestOutput().At(0).AssertText(
		"Disk: 100 KiB/s",
		"ignores invalid lines in diskstats")
}
