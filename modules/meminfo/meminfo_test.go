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

package meminfo

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
)

type meminfo map[string]int

func shouldReturn(info meminfo) {
	var out bytes.Buffer
	for key, value := range info {
		out.WriteString(fmt.Sprintf("%s:\t%d kB\n", key, value))
	}
	afero.WriteFile(fs, "/proc/meminfo", out.Bytes(), 0644)
}

func resetForTest() {
	currentInfo = base.ErrorValue{}
	once = sync.Once{}
	construct()
	// Flush upates for test.
	n := currentInfo.Next()
	update()
	<-n
}

func TestMeminfo(t *testing.T) {
	require := require.New(t)
	fs = afero.NewMemMapFs()
	shouldReturn(meminfo{
		"MemAvailable": 2048,
		"MemTotal":     4096,
		"MemFree":      1024,
	})
	testBar.New(t)
	resetForTest()

	def := New()
	avail := New()
	free := New()

	testBar.Run(def, avail, free)
	testBar.LatestOutput().Expect("on start")

	avail.Output(func(i Info) bar.Output {
		return outputs.Textf("%v", i.Available().Kibibytes())
	})
	free.Output(func(i Info) bar.Output {
		return outputs.Textf("%v", i.FreeFrac("Mem"))
	})
	testBar.LatestOutput(1, 2).AssertText(
		[]string{"Mem: 2.0 MiB", "2048", "0.25"}, "on start")

	shouldReturn(meminfo{
		"MemAvailable": 1024,
		"MemTotal":     4096,
		"MemFree":      256,
		"Cached":       512,
	})
	testBar.Tick()

	testBar.LatestOutput().AssertText(
		[]string{"Mem: 1.0 MiB", "1024", "0.0625"}, "on tick")

	shouldReturn(meminfo{
		"Cached":   1024,
		"Buffers":  512,
		"MemTotal": 4096,
		"MemFree":  512,
	})
	testBar.Tick()

	testBar.LatestOutput().AssertText(
		[]string{"Mem: 2.0 MiB", "2048", "0.125"}, "on tick")

	def.Output(func(i Info) bar.Output {
		return outputs.Textf("%v", i["Buffers"].Mebibytes())
	})
	testBar.LatestOutput(0).AssertText(
		[]string{"0.5", "2048", "0.125"}, "on template change")

	beforeTick := timing.Now()
	RefreshInterval(time.Minute)
	testBar.Tick()
	require.Equal(time.Minute, timing.Now().Sub(beforeTick), "RefreshInterval change")

	testBar.LatestOutput().Expect("on tick after refresh interval change")
}

func TestErrors(t *testing.T) {
	fs = afero.NewMemMapFs()
	testBar.New(t)
	resetForTest()

	availFrac := New()
	free := New()
	total := New()

	testBar.Run(availFrac, free, total)
	testBar.LatestOutput().AssertError("on start if missing meminfo")

	availFrac.Output(func(i Info) bar.Output {
		return outputs.Textf("%v", i.AvailFrac())
	})
	free.Output(func(i Info) bar.Output {
		memFree, ok := i["MemFree"]
		if !ok {
			return outputs.Errorf("Missing MemFree")
		}
		return outputs.Text(outputs.IBytesize(memFree))
	})
	total.Output(func(i Info) bar.Output {
		memTotal, ok := i["MemTotal"]
		if !ok {
			return outputs.Errorf("Missing MemTotal")
		}
		return outputs.Text(outputs.Bytesize(memTotal))
	})
	testBar.LatestOutput().Expect("template")

	afero.WriteFile(fs, "/proc/meminfo", []byte(`
	-- Lines in weird formats --
	Empty:

	Valid line:
	MemAvailable: 1024 kB
	Missing colon:
	MemFree 1024 kB
	Fields are non-numeric:
	MemTotal: ABCD kB
	`), 0644)
	testBar.Tick()
	out := testBar.LatestOutput()
	out.At(1).AssertError("non-numeric value")
	out.At(2).AssertError("non-numeric value")
	// MemAvailable is parsed, but total is 0.
	out.At(0).AssertText("+Inf")

	afero.WriteFile(fs, "/proc/meminfo", []byte(`
	MemAvailable: 1024 kB
	MemFree: 1024 kB
	MemTotal: 2048 kB
	`), 0644)
	testBar.Tick()
	testBar.LatestOutput().AssertText(
		[]string{"0.5", "1.0 MiB", "2.1 MB"},
		"when meminfo is back to normal")
}
