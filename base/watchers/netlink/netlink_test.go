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

package netlink

import (
	"errors"
	"net"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errFoo = errors.New("foo")
var someHwAddr = net.HardwareAddr{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd}

func reset() {
	TestMode()
	once = sync.Once{}
}

func assertUpdated(t *testing.T, ch <-chan Link, msgAndArgs ...interface{}) Link {
	select {
	case l := <-ch:
		return l
	case <-time.After(time.Second):
		assert.Fail(t, "Did not receive an update", msgAndArgs...)
		return Link{}
	}
}

func assertNoUpdate(t *testing.T, ch <-chan Link, msgAndArgs ...interface{}) {
	select {
	case _, ok := <-ch:
		if ok {
			assert.Fail(t, "Unexpected update", msgAndArgs...)
		}
	case <-time.After(time.Millisecond):
	}
}

func devNull(ch <-chan Link) {
	for range ch {
	}
}

func assertLinkEqual(t *testing.T, expected, actual Link, msgAndArgs ...interface{}) {
	if len(expected.HardwareAddr) == 0 {
		expected.HardwareAddr = nil
	}
	if len(actual.HardwareAddr) == 0 {
		actual.HardwareAddr = nil
	}
	assert.Equal(t, expected, actual, msgAndArgs...)
}

func TestErrors(t *testing.T) {
	reset()
	setInitialData(testNlRequest{err: errFoo}, testNlRequest{err: errFoo})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		assert.Fail(t, "Should not call subscribe")
		return nil, nil
	})
	sub := All()
	assertNoUpdate(t, sub, "on error during initial data")

	reset()
	setInitialData(testNlRequest{}, testNlRequest{err: errFoo})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		assert.Fail(t, "Should not call subscribe")
		return nil, nil
	})
	sub = All()
	assertNoUpdate(t, sub, "on error during initial data")

	reset()
	setInitialData(testNlRequest{}, testNlRequest{})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		return nil, errFoo
	})
	sub = All()
	assertNoUpdate(t, sub, "on error during subscribe")
}

func TestInitialData(t *testing.T) {
	reset()
	setInitialData(testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewLink(1, Link{Name: "lo1", State: Unknown}),
			msgNewLink(2, Link{Name: "wlan0", State: Up}),
			msgNewLink(3, Link{Name: "eno1", State: Dormant}),
		},
	}, testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewAddrs(1, net.IPv4(127, 0, 0, 1), nil),
			msgNewAddrs(2, net.IPv4(192, 168, 1, 0), net.IPv4(192, 168, 0, 1)),
			msgNewAddrs(2, net.IPv4(192, 168, 45, 0), net.IPv4(192, 168, 45, 1)),
			msgNewAddrs(5, net.IPv4(127, 0, 1, 1), nil),
		},
	})

	initialUpdates := map[string]Link{}
	sub := All()
	// one update for each link.
	for i := 0; i < 3; i++ {
		u := assertUpdated(t, sub, "intial data #%d", i)
		initialUpdates[u.Name] = u
	}
	assertNoUpdate(t, sub, "Only one update per link in initial data")

	assertLinkEqual(t, initialUpdates["lo1"], Link{
		Name:  "lo1",
		State: Unknown,
		IPs:   []net.IP{net.IPv4(127, 0, 0, 1)},
	})
	assertLinkEqual(t, initialUpdates["eno1"], Link{
		Name:  "eno1",
		State: Dormant,
	})
	assertLinkEqual(t, initialUpdates["wlan0"], Link{
		Name:  "wlan0",
		State: Up,
		IPs:   []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 45, 1)},
	})
}

func TestUpdates(t *testing.T) {
	reset()
	setInitialData(testNlRequest{}, testNlRequest{})
	msgCh, errCh := returnTestSubscriber()
	eno1 := Link{Name: "eno1", State: Unknown}

	sub := All()
	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Unknown})

	u := assertUpdated(t, sub, "Receives update of new link")
	assertLinkEqual(t, eno1, u)

	errCh <- errFoo
	assertNoUpdate(t, sub, "on error in Receive")

	msgCh <- msgNewAddrs(1, net.IPv4(192, 168, 0, 1), nil)
	u = assertUpdated(t, sub, "receives update after error")
	eno1.IPs = []net.IP{net.IPv4(192, 168, 0, 1)}
	assertLinkEqual(t, eno1, u, "IP is added and entire link is sent")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant})
	u = assertUpdated(t, sub)
	eno1.State = Dormant
	assertLinkEqual(t, eno1, u, "IP is not lost on link update")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant})
	assertNoUpdate(t, sub, "when nothing of interest changes")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: someHwAddr})
	u = assertUpdated(t, sub, "on gaining a hardware address")
	eno1.HardwareAddr = someHwAddr
	assertLinkEqual(t, eno1, u, "hardware address")

	msgCh <- msgNewLink(1, Link{Name: "eth0", State: Dormant, HardwareAddr: someHwAddr})
	u = assertUpdated(t, sub, "on rename")
	assertLinkEqual(t, Link{Name: "eno1", State: Gone}, u, "old link is gone")
	eno1.Name = "eth0"
	u = assertUpdated(t, sub, "on rename")
	assertLinkEqual(t, eno1, u, "link is renamed")

	sub2 := All()
	u = assertUpdated(t, sub2, "New subscriber updates on start")
	assertLinkEqual(t, eno1, u, "update has current information")

	Unsubscribe(sub2)

	msgCh <- msgNewAddrs(1, net.IPv4(192, 168, 0, 1), nil)
	assertNoUpdate(t, sub, "on adding same IP")
	assertNoUpdate(t, sub2, "after unsubscribe")

	msgCh <- msgNewAddrs(1, net.IPv4(10, 0, 0, 1), nil)
	u = assertUpdated(t, sub, "on adding different IP")
	eno1.IPs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(192, 168, 0, 1)}
	assertLinkEqual(t, eno1, u, "IP is added and entire link is sent")

	msgCh <- msgDelAddrs(1, net.IPv4(192, 168, 10, 1), nil)
	assertNoUpdate(t, sub, "on removing a non-existent IP")

	msgCh <- msgNewAddrs(3, net.IPv4(192, 168, 1, 1), nil)
	assertNoUpdate(t, sub, "on adding IP to non-existent link")

	msgCh <- msgDelAddrs(3, net.IPv4(192, 168, 1, 1), nil)
	assertNoUpdate(t, sub, "on removing IP from non-existent link")

	msgCh <- msgNewAddrs(1, net.IPv4(0, 0, 0, 0), nil)
	assertUpdated(t, sub)
	assertNoUpdate(t, sub2, "after unsubscribe")

	msgCh <- msgNewAddrs(1, net.IPv6loopback, nil)
	assertUpdated(t, sub)

	msgCh <- msgDelAddrs(1, net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 0, 1))
	u = assertUpdated(t, sub, "on removing an IP")
	eno1.IPs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback, net.IPv4(0, 0, 0, 0)}
	assertLinkEqual(t, eno1, u, "All other link information is preserved")

	msgCh <- msgDelLink(3, Link{Name: "wlan0"})
	assertNoUpdate(t, sub, "on removing non-existent link")

	msgCh <- msgDelLink(1, Link{})
	u = assertUpdated(t, sub, "on deleting link")
	assertLinkEqual(t, Link{Name: "eth0", State: Gone}, u)

	sub3 := All()
	assertNoUpdate(t, sub3, "when no links are present")
}

func TestFiltering(t *testing.T) {
	reset()
	setInitialData(testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewLink(1, Link{Name: "lo1", State: Unknown}),
			msgNewLink(2, Link{Name: "wlan0", State: Up}),
			msgNewLink(3, Link{Name: "eno1", State: Dormant}),
			msgNewLink(4, Link{Name: "wwan0", State: Down}),
		},
	}, testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewAddrs(1, net.IPv4(127, 0, 0, 1), nil),
			msgNewAddrs(2, net.IPv4(192, 168, 0, 1), nil),
			msgNewAddrs(2, net.IPv4(192, 168, 0, 2), nil),
		},
	})

	lo1 := Link{
		Name: "lo1",
		IPs:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	wlan0 := Link{
		Name:  "wlan0",
		State: Up,
		IPs:   []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 0, 2)},
	}
	wwan0 := Link{Name: "wwan0", State: Down}

	subW := WithPrefix("w")
	subLocal := ByName("lo1")
	subEth := ByName("eth0")
	subAll := All()

	for i := 0; i < 2; i++ {
		u := assertUpdated(t, subW, "Prefix('w') #%d", i)
		if u.Name == "wwan0" {
			assertLinkEqual(t, wwan0, u)
		} else {
			assertLinkEqual(t, wlan0, u)
		}
	}
	assertNoUpdate(t, subW, "Only two links start with 'w'")

	u := assertUpdated(t, subLocal, "Name('lo1')")
	assertLinkEqual(t, lo1, u)
	assertNoUpdate(t, subLocal, "Only one update for named link")

	assertNoUpdate(t, subEth, "No update for named link not found")

	for i := 0; i < 4; i++ {
		assertUpdated(t, subAll, "All() #%d", i)
	}
}

func TestTestMode(t *testing.T) {
	nlt := TestMode()

	subEth := ByName("eth0")
	subAll := All()

	id := nlt.AddLink(Link{Name: "eno1"})
	assertUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	nlt.AddIP(id, net.IPv4(10, 0, 0, 1))
	assertUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	nlt.RemoveIP(id, net.IPv4(10, 0, 0, 1))
	assertUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	nlt.RemoveLink(id)
	assertUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	id = nlt.AddLink(Link{Name: "eth0"})
	assertUpdated(t, subAll)
	assertUpdated(t, subEth)

	nlt.AddIP(id, net.IPv4(10, 0, 0, 1))
	nlt.AddIP(id, net.IPv4(10, 0, 2, 1))
	nlt.AddIP(id, net.IPv6loopback)
	for i := 0; i < 3; i++ {
		assertUpdated(t, subAll)
		assertUpdated(t, subEth)
	}

	nlt.RemoveIP(id, net.IPv4(10, 0, 2, 1))
	expected := Link{
		Name: "eth0",
		IPs:  []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback},
	}
	u := assertUpdated(t, subEth)
	assertLinkEqual(t, expected, u)
	u = assertUpdated(t, subAll)
	assertLinkEqual(t, expected, u)

	nlt.UpdateLink(id, Link{Name: "eth0", State: Up})
	expected.State = Up
	u = assertUpdated(t, subEth)
	assertLinkEqual(t, expected, u)
	assertUpdated(t, subAll)
}
