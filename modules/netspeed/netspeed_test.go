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

	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
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

var signalChan = make(chan struct{})

func init() {
	linkByName = func(name string) (netlink.Link, error) {
		ifacesLock.Lock()
		link, ok := ifaces[name]
		ifacesLock.Unlock()
		select {
		case signalChan <- struct{}{}:
		default:
		}
		if !ok {
			return nil, fmt.Errorf("No such link: %s", name)
		}
		return link, nil
	}
}

func TestNetspeed(t *testing.T) {
	assert := assert.New(t)
	testBar.New(t)

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 1024,
		TxBytes: 1024,
	})

	n := New("if0").
		RefreshInterval(time.Second).
		OutputTemplate(`{{.Rx.KibibytesPerSecond}}/{{.Tx.KibibytesPerSecond}}`)

	testBar.Run(n)
	testBar.AssertNoOutput("on start")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 4096,
		TxBytes: 2048,
	})
	testBar.Tick()

	testBar.NextOutput().AssertEqual(outputs.Text("3/1"), "on tick")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 8192,
		TxBytes: 3072,
	})
	testBar.Tick()

	testBar.NextOutput().AssertEqual(outputs.Text("4/1"), "on tick")

	n.OutputTemplate(`{{.Total | ibyterate}}`)
	testBar.NextOutput().AssertEqual(
		outputs.Text("5.0 KiB/s"),
		"uses previous result on output function change")

	n.OutputTemplate(`{{.Total | byterate}}`)
	testBar.NextOutput().AssertEqual(
		outputs.Text("5.1 kB/s"),
		"uses previous result on output function change")
	testBar.Tick()
	testBar.NextOutput().AssertEqual(
		outputs.Text("0 B/s"), "on tick after output function change")

	beforeTick := timing.Now()
	n.RefreshInterval(time.Minute)
	testBar.Tick()
	assert.Equal(time.Minute, timing.Now().Sub(beforeTick),
		"RefreshInterval change")
	testBar.NextOutput().Expect("RefreshInterval change")
}

func TestErrors(t *testing.T) {
	testBar.New(t)

	removeLink("if0")
	n := New("if0").RefreshInterval(time.Second)
	testBar.Run(n)
	testBar.NextOutput().AssertError("on start for missing interface")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 0,
		TxBytes: 0,
	})
	go testBar.Click(0)
	<-signalChan
	testBar.NextOutput().AssertText([]string{},
		"clears error on click after interface is available")

	setLink("if0", netlink.LinkStatistics{
		RxBytes: 4096,
		TxBytes: 2048,
	})
	testBar.Tick()
	testBar.NextOutput().AssertText(
		[]string{"2.0 KiB/s up | 4.0 KiB/s down"}, "on tick")

	removeLink("if0")
	testBar.Tick()
	testBar.NextOutput().AssertError("on tick after losing interface")
}
