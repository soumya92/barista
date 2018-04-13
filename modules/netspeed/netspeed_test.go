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

package netspeed

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"
	"github.com/vishvananda/netlink"

	_ "github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

type testLink netlink.LinkStatistics

func (t testLink) Attrs() *netlink.LinkAttrs {
	return &netlink.LinkAttrs{Statistics: (*netlink.LinkStatistics)(&t)}
}
func (t testLink) Type() string { return "test" }

var ifaces = make(map[string]testLink)
var ifacesLock sync.Mutex

func removeLink(name string) {
	ifacesLock.Lock()
	defer ifacesLock.Unlock()
	delete(ifaces, name)
}

func setLink(name string, stats netlink.LinkStatistics) {
	ifacesLock.Lock()
	defer ifacesLock.Unlock()
	ifaces[name] = testLink(stats)
}

func init() {
	linkByName = func(name string) (netlink.Link, error) {
		ifacesLock.Lock()
		link, ok := ifaces[name]
		ifacesLock.Unlock()
		if !ok {
			return nil, fmt.Errorf("No such link: %s", name)
		}
		return link, nil
	}
}

func TestNetspeed(t *testing.T) {
	assert := assert.New(t)
	scheduler.TestMode(true)

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 1024,
		TxBytes: 1024,
	})

	n := New("if0").
		RefreshInterval(time.Second).
		OutputTemplate(outputs.TextTemplate(
			`{{.Rx.KibibytesPerSecond}}/{{.Tx.KibibytesPerSecond}}`))

	tester := testModule.NewOutputTester(t, n)
	tester.AssertNoOutput("on start")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 4096,
		TxBytes: 2048,
	})
	scheduler.NextTick()

	tester.AssertOutputEquals(outputs.Text("3/1"), "on tick")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 8192,
		TxBytes: 3072,
	})
	scheduler.NextTick()

	tester.AssertOutputEquals(outputs.Text("4/1"), "on tick")

	n.OutputTemplate(outputs.TextTemplate(`{{.Total | ibyterate}}`))
	tester.AssertOutputEquals(
		outputs.Text("5.0 KiB/s"),
		"uses previous result on output function change")

	n.OutputTemplate(outputs.TextTemplate(`{{.Total | byterate}}`))
	tester.AssertOutputEquals(
		outputs.Text("5.1 kB/s"),
		"uses previous result on output function change")

	scheduler.NextTick()
	tester.AssertOutputEquals(
		outputs.Text("0 B/s"), "on tick after output function change")

	beforeTick := scheduler.Now()
	n.RefreshInterval(time.Minute)
	scheduler.NextTick()
	assert.Equal(time.Minute, scheduler.Now().Sub(beforeTick),
		"RefreshInterval change")

	tester.Drain()
}

func TestErrors(t *testing.T) {
	scheduler.TestMode(true)

	removeLink("if0")
	n := New("if0").
		RefreshInterval(time.Second).
		OutputTemplate(outputs.TextTemplate(
			`{{.Rx.KibibytesPerSecond}}/{{.Tx.KibibytesPerSecond}}`))
	tester := testModule.NewOutputTester(t, n)
	tester.AssertError("on start for missing interface")

	scheduler.NextTick()
	tester.AssertError("after tick with missing interface")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 0,
		TxBytes: 0,
	})
	scheduler.NextTick()
	tester.AssertNoOutput("first tick after interface is available")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 4096,
		TxBytes: 2048,
	})
	scheduler.NextTick()
	tester.AssertOutputEquals(outputs.Text("4/2"), "on tick")
}
