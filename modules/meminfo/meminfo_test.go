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
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchrcom/testify/assert"

	_ "github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
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
	scheduler.TestMode(true)

	shouldReturn(meminfo{
		"MemAvailable": 2048,
		"MemTotal":     4096,
		"MemFree":      1024,
	})

	m := New()
	avail := m.OutputTemplate(outputs.TextTemplate(`{{.Available.In "KiB"}}`))

	tester1 := testModule.NewOutputTester(t, avail)
	out := tester1.AssertOutput("on start")
	assert.Equal(outputs.Text("2048"), out)

	shouldReturn(meminfo{
		"MemAvailable": 1024,
		"MemTotal":     4096,
		"MemFree":      256,
		"Cached":       512,
	})
	scheduler.NextTick()

	out = tester1.AssertOutput("on tick")
	assert.Equal(outputs.Text("1024"), out)

	free := m.OutputTemplate(outputs.TextTemplate(`{{.FreeFrac "Mem"}}`))

	tester2 := testModule.NewOutputTester(t, free)
	out = tester2.AssertOutput("on start")
	assert.Equal(outputs.Text("0.0625"), out)

	tester1.Drain()

	shouldReturn(meminfo{
		"Cached":   1024,
		"Buffers":  512,
		"MemTotal": 4096,
		"MemFree":  512,
	})
	scheduler.NextTick()

	out = tester1.AssertOutput("on tick")
	assert.Equal(outputs.Text("2048"), out)

	out = tester2.AssertOutput("on tick")
	assert.Equal(outputs.Text("0.125"), out)

	beforeTick := scheduler.Now()
	m.RefreshInterval(time.Minute)
	scheduler.NextTick()
	assert.Equal(time.Minute, scheduler.Now().Sub(beforeTick), "RefreshInterval change")

	tester1.Drain()
	tester2.Drain()
}

func TestErrors(t *testing.T) {
	fs = afero.NewMemMapFs()
	scheduler.TestMode(true)

	m := New()

	availFrac := m.OutputTemplate(outputs.TextTemplate(`{{.AvailFrac}}`))
	free := m.OutputTemplate(outputs.TextTemplate(`{{.MemFree.IEC}}`))
	total := m.OutputTemplate(outputs.TextTemplate(`{{.MemTotal.SI}}`))

	tester := testModule.NewOutputTester(t, availFrac)
	tester1 := testModule.NewOutputTester(t, free)
	tester2 := testModule.NewOutputTester(t, total)

	tester.AssertError("on start if missing meminfo")
	tester1.AssertError("on start if missing meminfo")
	tester2.AssertError("on start if missing meminfo")

	tester.Drain()
	tester1.Drain()
	tester2.Drain()

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

	scheduler.NextTick()
	tester.AssertError("non-numeric value")
	tester1.AssertError("non-numeric value")
	tester2.AssertError("non-numeric value")

	afero.WriteFile(fs, "/proc/meminfo", []byte(`
	MemAvailable: 1024 kB
	MemFree: 1024 kB
	MemTotal: 2048 kB
	`), 0644)

	scheduler.NextTick()

	out := tester.AssertOutput("when meminfo is back to normal")
	assert.Equal(t, outputs.Textf("0.5"), out)

	out = tester1.AssertOutput("when meminfo is back to normal")
	assert.Equal(t, outputs.Textf("1.0 MiB"), out)

	out = tester2.AssertOutput("when meminfo is back to normal")
	assert.Equal(t, outputs.Textf("2.1 MB"), out)
}

// TODO: Remove this and spec out a "units" package.
func TestInvalidBaseInParse(t *testing.T) {
	fs = afero.NewMemMapFs()
	shouldReturn(meminfo{"MemTotal": 1})

	m := New()
	submodule := m.OutputTemplate(outputs.TextTemplate(`{{.MemTotal.In "foo"}}`))
	tester := testModule.NewOutputTester(t, submodule)
	out := tester.AssertOutput("on start")
	assert.Equal(t, outputs.Text("1024"), out)
}
