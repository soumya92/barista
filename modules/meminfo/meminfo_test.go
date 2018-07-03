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
	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/base"
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

func TestMeminfo(t *testing.T) {
	assert := assert.New(t)
	fs = afero.NewMemMapFs()
	testBar.New(t)
	currentInfo = base.ErrorValue{}
	once = sync.Once{}

	shouldReturn(meminfo{
		"MemAvailable": 2048,
		"MemTotal":     4096,
		"MemFree":      1024,
	})

	avail := New().OutputTemplate(`{{.Available.Kibibytes}}`)
	free := New().OutputTemplate(`{{.FreeFrac "Mem"}}`)

	testBar.Run(avail, free)
	testBar.LatestOutput().AssertText(
		[]string{"2048", "0.25"}, "on start")

	shouldReturn(meminfo{
		"MemAvailable": 1024,
		"MemTotal":     4096,
		"MemFree":      256,
		"Cached":       512,
	})
	testBar.Tick()

	testBar.LatestOutput().AssertText(
		[]string{"1024", "0.0625"}, "on tick")

	shouldReturn(meminfo{
		"Cached":   1024,
		"Buffers":  512,
		"MemTotal": 4096,
		"MemFree":  512,
	})
	testBar.Tick()

	testBar.LatestOutput().AssertText(
		[]string{"2048", "0.125"}, "on tick")

	beforeTick := timing.Now()
	RefreshInterval(time.Minute)
	testBar.Tick()
	assert.Equal(time.Minute, timing.Now().Sub(beforeTick), "RefreshInterval change")

	testBar.LatestOutput().Expect("on refresh interval change")
}

func TestErrors(t *testing.T) {
	fs = afero.NewMemMapFs()
	testBar.New(t)
	currentInfo = base.ErrorValue{}
	once = sync.Once{}

	availFrac := New().OutputTemplate(`{{.AvailFrac}}`)
	free := New().OutputTemplate(`{{.MemFree | ibytesize}}`)
	total := New().OutputTemplate(`{{.MemTotal | bytesize}}`)

	testBar.Run(availFrac, free, total)
	testBar.LatestOutput().AssertError("on start if missing meminfo")

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
