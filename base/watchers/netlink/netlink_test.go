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
	"reflect"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var errFoo = errors.New("foo")
var hwA = []net.HardwareAddr{
	{0xd7, 0x05, 0x9f, 0xb2, 0x56, 0x16},
	{0x0b, 0x00, 0xe7, 0x2d, 0xe7, 0x97},
	{0x7d, 0xb1, 0x55, 0x94, 0xd0, 0xdf},
	{0x5c, 0xf3, 0xe0, 0xaf, 0x67, 0xbc},
	{0x24, 0x04, 0xe7, 0x2e, 0xa5, 0xf1},
	{0x67, 0xff, 0x3e, 0x65, 0xf3, 0xc7},
	{0x34, 0x7d, 0x8a, 0xbc, 0xce, 0x51},
	{0x12, 0x3d, 0x6e, 0x5a, 0xbb, 0x42},
	{0xb5, 0x10, 0x10, 0x81, 0xc6, 0x87},
	{0xaf, 0x04, 0xd3, 0x98, 0x9c, 0xd9},
}

func reset() {
	TestMode()
	once = sync.Once{}
}

func assertUpdated(t *testing.T, s Subscription, msgAndArgs ...interface{}) Link {
	select {
	case l := <-s:
		return l
	case <-time.After(time.Second):
		require.Fail(t, "Did not receive an update", msgAndArgs...)
		return Link{}
	}
}

func assertMultiUpdated(t *testing.T, m MultiSubscription, msgAndArgs ...interface{}) []Link {
	select {
	case l := <-m:
		return l
	case <-time.After(time.Second):
		require.Fail(t, "Did not receive an update", msgAndArgs...)
		return nil
	}
}

func assertNoUpdate(t *testing.T, sub interface{}, msgAndArgs ...interface{}) {
	cases := []reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub)},
		{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(time.After(time.Millisecond))},
	}
	chosen, _, ok := reflect.Select(cases)
	if chosen == 0 && ok {
		require.Fail(t, "Unexpected update", msgAndArgs...)
	}
}

func TestErrors(t *testing.T) {
	reset()
	setInitialData(testNlRequest{err: errFoo}, testNlRequest{err: errFoo})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		require.Fail(t, "Should not call subscribe")
		return nil, nil
	})
	sub := All()
	links := assertMultiUpdated(t, sub, "on error during initial data")
	require.Empty(t, links, "no links on error")

	reset()
	setInitialData(testNlRequest{}, testNlRequest{err: errFoo})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		require.Fail(t, "Should not call subscribe")
		return nil, nil
	})
	sub = All()
	links = assertMultiUpdated(t, sub, "on error during initial data")
	require.Empty(t, links, "no links on error")

	reset()
	setInitialData(testNlRequest{}, testNlRequest{})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		return nil, errFoo
	})
	sub = All()
	links = assertMultiUpdated(t, sub, "on error during subscribe")
	require.Empty(t, links, "no links on error")
}

func TestInitialData(t *testing.T) {
	reset()
	setInitialData(testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewLink(1, Link{Name: "lo1", State: Unknown, HardwareAddr: hwA[1]}),
			msgNewLink(2, Link{Name: "wlan0", State: Up, HardwareAddr: hwA[2]}),
			msgNewLink(3, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[3]}),
		},
	}, testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewAddrs(1, net.IPv4(127, 0, 0, 1), nil),
			msgNewAddrs(2, net.IPv4(192, 168, 1, 0), net.IPv4(192, 168, 0, 1)),
			msgNewAddrs(2, net.IPv4(192, 168, 45, 0), net.IPv4(192, 168, 45, 1)),
			msgNewAddrs(5, net.IPv4(127, 0, 1, 1), nil),
		},
	})

	sub := All()
	initialUpdate := assertMultiUpdated(t, sub, "intial data")
	assertNoUpdate(t, sub, "Only one initial update")

	require.Equal(t, []Link{
		{
			Name:         "wlan0",
			State:        Up,
			HardwareAddr: hwA[2],
			IPs:          []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 45, 1)},
		},
		{
			Name:         "eno1",
			State:        Dormant,
			HardwareAddr: hwA[3],
		},
		{
			Name:         "lo1",
			State:        Unknown,
			HardwareAddr: hwA[1],
			IPs:          []net.IP{net.IPv4(127, 0, 0, 1)},
		},
	}, initialUpdate)
}

func TestUpdates(t *testing.T) {
	reset()
	setInitialData(testNlRequest{}, testNlRequest{})
	msgCh, errCh := returnTestSubscriber()
	eno1 := Link{Name: "eno1", State: Unknown, HardwareAddr: hwA[4]}

	sub := All()
	assertMultiUpdated(t, sub, "on start")
	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Unknown, HardwareAddr: hwA[4]})

	links := assertMultiUpdated(t, sub, "Receives update of new link")
	require.Equal(t, []Link{eno1}, links)

	errCh <- errFoo
	assertNoUpdate(t, sub, "on error in Receive")

	msgCh <- msgNewAddrs(1, net.IPv4(192, 168, 0, 1), nil)
	links = assertMultiUpdated(t, sub, "receives update after error")
	eno1.IPs = []net.IP{net.IPv4(192, 168, 0, 1)}
	require.Equal(t, []Link{eno1}, links, "IP is added and entire link is sent")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[4]})
	links = assertMultiUpdated(t, sub)
	eno1.State = Dormant
	require.Equal(t, []Link{eno1}, links, "IP is not lost on link update")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[4]})
	assertNoUpdate(t, sub, "when nothing of interest changes")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[0]})
	links = assertMultiUpdated(t, sub, "on gaining a hardware address")
	eno1.HardwareAddr = hwA[0]
	require.Equal(t, []Link{eno1}, links, "hardware address")

	msgCh <- msgNewLink(1, Link{Name: "eth0", State: Dormant, HardwareAddr: hwA[0]})
	eno1.Name = "eth0"
	links = assertMultiUpdated(t, sub, "on rename")
	require.Equal(t, []Link{eno1}, links, "link is renamed")

	sub2 := All()
	links = assertMultiUpdated(t, sub2, "New subscriber updates on start")
	require.Equal(t, []Link{eno1}, links, "update has current information")

	sub2.Unsubscribe()

	msgCh <- msgNewAddrs(1, net.IPv4(192, 168, 0, 1), nil)
	assertNoUpdate(t, sub, "on adding same IP")
	assertNoUpdate(t, sub2, "after unsubscribe")

	msgCh <- msgNewAddrs(1, net.IPv4(10, 0, 0, 1), nil)
	links = assertMultiUpdated(t, sub, "on adding different IP")
	eno1.IPs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(192, 168, 0, 1)}
	require.Equal(t, []Link{eno1}, links, "IP is added and entire link is sent")

	msgCh <- msgDelAddrs(1, net.IPv4(192, 168, 10, 1), nil)
	assertNoUpdate(t, sub, "on removing a non-existent IP")

	msgCh <- msgNewAddrs(3, net.IPv4(192, 168, 1, 1), nil)
	assertNoUpdate(t, sub, "on adding IP to non-existent link")

	msgCh <- msgDelAddrs(3, net.IPv4(192, 168, 1, 1), nil)
	assertNoUpdate(t, sub, "on removing IP from non-existent link")

	msgCh <- msgNewAddrs(1, net.IPv4(0, 0, 0, 0), nil)
	assertMultiUpdated(t, sub)
	assertNoUpdate(t, sub2, "after unsubscribe")

	msgCh <- msgNewAddrs(1, net.IPv6loopback, nil)
	assertMultiUpdated(t, sub)

	msgCh <- msgDelAddrs(1, net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 0, 1))
	links = assertMultiUpdated(t, sub, "on removing an IP")
	eno1.IPs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback, net.IPv4(0, 0, 0, 0)}
	require.Equal(t, []Link{eno1}, links, "All other link information is preserved")

	msgCh <- msgDelLink(3, Link{Name: "wlan0"})
	assertNoUpdate(t, sub, "on removing non-existent link")

	msgCh <- msgDelLink(1, Link{})
	links = assertMultiUpdated(t, sub, "on deleting link")
	require.Empty(t, links, "all links are gone")

	sub3 := All()
	links = assertMultiUpdated(t, sub3, "on first subscribe")
	require.Empty(t, links, "when no links are present")
}

func TestFiltering(t *testing.T) {
	reset()
	setInitialData(testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewLink(1, Link{Name: "lo1", State: Unknown, HardwareAddr: hwA[4]}),
			msgNewLink(2, Link{Name: "wlan0", State: Up, HardwareAddr: hwA[5]}),
			msgNewLink(3, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[6]}),
			msgNewLink(4, Link{Name: "wwan0", State: Down, HardwareAddr: hwA[7]}),
		},
	}, testNlRequest{
		msgs: []syscall.NetlinkMessage{
			msgNewAddrs(1, net.IPv4(127, 0, 0, 1), nil),
			msgNewAddrs(2, net.IPv4(192, 168, 0, 1), nil),
			msgNewAddrs(2, net.IPv4(192, 168, 0, 2), nil),
		},
	})
	msgCh, _ := returnTestSubscriber()

	lo1 := Link{
		Name:         "lo1",
		IPs:          []net.IP{net.IPv4(127, 0, 0, 1)},
		HardwareAddr: hwA[4],
	}
	wlan0 := Link{
		Name:         "wlan0",
		State:        Up,
		IPs:          []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 0, 2)},
		HardwareAddr: hwA[5],
	}
	wwan0 := Link{Name: "wwan0", State: Down, HardwareAddr: hwA[7]}

	subW := WithPrefix("w")
	subLocal := ByName("lo1")
	subEth := ByName("eth0")
	subAll := All()
	subAny := Any()

	link := assertUpdated(t, subW, "Prefix('w')")
	require.Equal(t, wlan0, link)
	assertNoUpdate(t, subW, "Only one update for prefixed link")

	link = assertUpdated(t, subLocal, "Name('lo1')")
	require.Equal(t, lo1, link)
	assertNoUpdate(t, subLocal, "Only one update for named link")

	link = assertUpdated(t, subEth, "No update for named link not found")
	require.Equal(t, Link{State: Gone}, link)
	assertNoUpdate(t, subEth, "Only one update on start")

	links := assertMultiUpdated(t, subAll, "All()")
	require.Equal(t, 4, len(links), "All four links in update to All()")
	assertNoUpdate(t, subAll, "Only one update for All()")

	link = assertUpdated(t, subAny, "On start")
	require.Equal(t, wlan0, link, "Best link is sent to Any()")
	assertNoUpdate(t, subAny, "Only one update for Any()")

	msgCh <- msgNewLink(1, Link{Name: "lo1", State: Up, HardwareAddr: hwA[4]})
	assertUpdated(t, subLocal, "Named link changed")
	assertNoUpdate(t, subEth, "Named link still not present")
	assertMultiUpdated(t, subAll, "A link changed")
	assertUpdated(t, subAny, "A link changed")
	assertNoUpdate(t, subW, "No relevant link changed")

	msgCh <- msgNewLink(1, Link{Name: "lo1", State: Down, HardwareAddr: hwA[4]})
	assertUpdated(t, subLocal, "Named link changed")
	assertNoUpdate(t, subEth, "Named link still not present")
	assertMultiUpdated(t, subAll, "A link changed")
	assertUpdated(t, subAny, "A link changed")
	assertNoUpdate(t, subW, "No relevant link changed")

	msgCh <- msgNewLink(4, Link{Name: "wwan0", State: Up, HardwareAddr: hwA[7]})
	assertNoUpdate(t, subLocal, "Named link unchanged")
	assertNoUpdate(t, subEth, "Named link still not present")
	assertMultiUpdated(t, subAll, "A link changed")
	assertUpdated(t, subAny, "A link changed")
	link = assertUpdated(t, subW, "Relevant link changed")
	require.Equal(t, wlan0, link, "'best' link is still the same")

	msgCh <- msgNewLink(2, Link{Name: "wlan0", State: Dormant, HardwareAddr: hwA[5]})
	link = assertUpdated(t, subW, "Relevant link changed")
	wwan0.State = Up
	require.Equal(t, wwan0, link, "Re-order on state change")
	link = assertUpdated(t, subAny, "A link changed")
	require.Equal(t, wwan0, link, "Re-order on state change")
}

func TestTestMode(t *testing.T) {
	nlt := TestMode()

	subEth := ByName("eth0")
	assertUpdated(t, subEth)
	subAll := All()
	assertMultiUpdated(t, subAll)

	id := nlt.AddLink(Link{Name: "eno1"})
	assertMultiUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	nlt.AddIP(id, net.IPv4(10, 0, 0, 1))
	assertMultiUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	nlt.RemoveIP(id, net.IPv4(10, 0, 0, 1))
	assertMultiUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	nlt.RemoveLink(id)
	assertMultiUpdated(t, subAll)
	assertNoUpdate(t, subEth)

	id = nlt.AddLink(Link{Name: "eth0", HardwareAddr: hwA[8]})
	assertMultiUpdated(t, subAll)
	assertUpdated(t, subEth)

	nlt.AddIP(id, net.IPv4(10, 0, 0, 1))
	nlt.AddIP(id, net.IPv4(10, 0, 2, 1))
	nlt.AddIP(id, net.IPv6loopback)
	for i := 0; i < 3; i++ {
		assertMultiUpdated(t, subAll)
		assertUpdated(t, subEth)
	}

	nlt.RemoveIP(id, net.IPv4(10, 0, 2, 1))
	expected := Link{
		Name:         "eth0",
		IPs:          []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback},
		HardwareAddr: hwA[8],
	}
	link := assertUpdated(t, subEth)
	require.Equal(t, expected, link)
	links := assertMultiUpdated(t, subAll)
	require.Equal(t, []Link{expected}, links)

	nlt.UpdateLink(id, Link{State: Up})
	expected.State = Up
	link = assertUpdated(t, subEth)
	require.Equal(t, expected, link)
	assertMultiUpdated(t, subAll)

	subEth.Unsubscribe()
	nlt.UpdateLink(id, Link{State: Dormant})
	assertNoUpdate(t, subEth, "after unsubscribe")

	nlt.UpdateLink(id, Link{HardwareAddr: hwA[9]})
	link = assertUpdated(t, Any())
	expected.State = Dormant
	expected.HardwareAddr = hwA[9]
	require.Equal(t, expected, link, "unset properties are retained")
}
