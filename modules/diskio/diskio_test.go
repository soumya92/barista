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
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
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
	afero.WriteFile(fs, "/proc/diskstats", out.Bytes(), 0644)
}

func TestDiskIo(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()
	scheduler.TestMode(true)

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{0, 0},
	})

	d := New()
	d.signalChan = make(chan bool)
	sda1 := d.Disk("sda1").OutputTemplate(outputs.TextTemplate(`{{.Total.In "b"}}`))

	tester1 := testModule.NewOutputTester(t, sda1)
	<-d.signalChan

	tester1.AssertNoOutput("on start")

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{9, 9},
	})
	scheduler.NextTick()
	<-d.signalChan

	out := tester1.AssertOutput("on tick")
	// 9+9 sectors / 3 seconds = 6 sectors / second * 512 bytes / sector = 3027 bytes.
	assert.Equal(outputs.Text("3072"), out)

	// Simpler math.
	d.RefreshInterval(time.Second)

	sdb1 := d.Disk("sdb1").OutputTemplate(outputs.TextTemplate(`{{.Total.IEC}}`))
	tester2 := testModule.NewOutputTester(t, sdb1)
	tester2.AssertNoOutput("on start")

	// Adding a new submodule causes updates to all other submodules.
	// TODO: See if this behaviour can be adjusted.
	tester1.Drain()
	drainChan(d.signalChan)

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{9, 10},
	})
	scheduler.NextTick()
	<-d.signalChan

	out = tester1.AssertOutput("on tick")
	assert.Equal(outputs.Text("512"), out)

	tester2.AssertNoOutput("for missing disk")

	shouldReturn(diskstats{
		"sda":  []int{0, 0},
		"sda1": []int{9, 10},
	})
	scheduler.NextTick()
	<-d.signalChan

	out = tester1.AssertOutput("on tick")
	assert.Equal(outputs.Text("0"), out)

	sda1.OutputFunc(func(i IO) bar.Output {
		return outputs.Textf("%s", i.Total().SI())
	})
	<-d.signalChan

	out = tester1.AssertOutput("on output func change")
	assert.Equal(outputs.Text("0 B"), out)

	tester2.AssertNoOutput("for missing disk")

	shouldReturn(diskstats{
		"sdb":  []int{0, 0},
		"sdb1": []int{300, 0},
	})
	scheduler.NextTick()
	<-d.signalChan

	out = tester1.AssertOutput("first tick after disk is removed")
	assert.Empty(out, "output is cleared when disk is removed")

	tester2.AssertNoOutput("first tick after disk is added")

	shouldReturn(diskstats{
		"sdb":  []int{0, 0},
		"sdb1": []int{300, 100},
	})
	scheduler.NextTick()
	<-d.signalChan

	tester1.AssertNoOutput("for missing disk")

	out = tester2.AssertOutput("on tick")
	assert.Equal(outputs.Text("50 KiB"), out)
}

func TestErrors(t *testing.T) {
	fs = afero.NewMemMapFs()
	scheduler.TestMode(true)

	d := New()
	d.signalChan = make(chan bool)

	sda := d.Disk("sda")
	sda1 := d.Disk("sda1")
	sda2 := d.Disk("sda2")

	tester := testModule.NewOutputTester(t, sda)
	tester1 := testModule.NewOutputTester(t, sda1)
	tester2 := testModule.NewOutputTester(t, sda2)

	tester.AssertError("on start if missing diskstats")
	tester1.AssertError("on start if missing diskstats")
	tester2.AssertError("on start if missing diskstats")

	// Adding modules causes extra updates, so swallow those.
	tester.Drain()
	tester1.Drain()
	tester2.Drain()
	drainChan(d.signalChan)

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

	scheduler.NextTick()
	<-d.signalChan
	tester1.AssertError("invalid read count")
	tester2.AssertError("invalid write count")
	// First tick initialises the stats,
	// but we won't have a delta until the next tick.
	tester.AssertNoOutput("on first tick")

	afero.WriteFile(fs, "/proc/diskstats", []byte(`
0 0 sda 0 0 400 0 0 0 400 0 0 0 0
`), 0644)

	scheduler.NextTick()
	<-d.signalChan

	out := tester.AssertOutput("on second tick")
	assert.Equal(t, outputs.Textf("Disk: 100 KiB/s"), out,
		"ignores invalid lines in diskstats")
}

func drainChan(ch <-chan bool) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
