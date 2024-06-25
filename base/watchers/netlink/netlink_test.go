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

	"github.com/soumya92/barista/testing/notifier"
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

type nexter interface {
	Next() <-chan struct{}
}

func assertUpdated(t *testing.T, next <-chan struct{}, sub nexter, msgAndArgs ...interface{}) <-chan struct{} {
	notifier.AssertClosed(t, next, msgAndArgs...)
	return sub.Next()
}

func TestErrors(t *testing.T) {
	reset()
	setInitialData(testNlRequest{err: errFoo}, testNlRequest{err: errFoo})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		require.Fail(t, "Should not call subscribe")
		return nil, nil
	})
	require.Empty(t, All().Get(), "no links on error")

	reset()
	setInitialData(testNlRequest{}, testNlRequest{err: errFoo})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		require.Fail(t, "Should not call subscribe")
		return nil, nil
	})
	require.Empty(t, All().Get(), "no links on error")

	reset()
	setInitialData(testNlRequest{}, testNlRequest{})
	returnCustomSubscriber(func(int, ...uint) (nlReceiver, error) {
		return nil, errFoo
	})
	require.Empty(t, All().Get(), "no links on error")
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
	next := sub.Next()
	notifier.AssertNoUpdate(t, next, "initial data populated on call to All()")

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
	}, sub.Get())
}

func TestUpdates(t *testing.T) {
	reset()
	setInitialData(testNlRequest{}, testNlRequest{})
	msgCh, errCh := returnTestSubscriber()
	eno1 := Link{Name: "eno1", State: Unknown, HardwareAddr: hwA[4]}

	sub := All()
	next := sub.Next()
	notifier.AssertNoUpdate(t, next, "on start")
	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Unknown, HardwareAddr: hwA[4]})

	next = assertUpdated(t, next, sub, "Receives update of new link")
	require.Equal(t, []Link{eno1}, sub.Get())

	errCh <- errFoo
	notifier.AssertNoUpdate(t, next, "on error in Receive")

	msgCh <- msgNewAddrs(1, net.IPv4(192, 168, 0, 1), nil)
	next = assertUpdated(t, next, sub, "receives update after error")
	eno1.IPs = []net.IP{net.IPv4(192, 168, 0, 1)}
	require.Equal(t, []Link{eno1}, sub.Get(), "IP is added and entire link is sent")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[4]})
	next = assertUpdated(t, next, sub)
	eno1.State = Dormant
	require.Equal(t, []Link{eno1}, sub.Get(), "IP is not lost on link update")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[4]})
	notifier.AssertNoUpdate(t, next, "when nothing of interest changes")

	msgCh <- msgNewLink(1, Link{Name: "eno1", State: Dormant, HardwareAddr: hwA[0]})
	next = assertUpdated(t, next, sub, "on gaining a hardware address")
	eno1.HardwareAddr = hwA[0]
	require.Equal(t, []Link{eno1}, sub.Get(), "hardware address")

	msgCh <- msgNewLink(1, Link{Name: "eth0", State: Dormant, HardwareAddr: hwA[0]})
	eno1.Name = "eth0"
	next = assertUpdated(t, next, sub, "on rename")
	require.Equal(t, []Link{eno1}, sub.Get(), "link is renamed")

	sub2 := All()
	require.Equal(t, []Link{eno1}, sub2.Get(), "update has current information")

	msgCh <- msgNewAddrs(1, net.IPv4(192, 168, 0, 1), nil)
	notifier.AssertNoUpdate(t, next, "on adding same IP")

	msgCh <- msgNewAddrs(1, net.IPv4(10, 0, 0, 1), nil)
	next = assertUpdated(t, next, sub, "on adding different IP")
	eno1.IPs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(192, 168, 0, 1)}
	require.Equal(t, []Link{eno1}, sub.Get(), "IP is added and entire link is sent")

	msgCh <- msgDelAddrs(1, net.IPv4(192, 168, 10, 1), nil)
	notifier.AssertNoUpdate(t, next, "on removing a non-existent IP")

	msgCh <- msgNewAddrs(3, net.IPv4(192, 168, 1, 1), nil)
	notifier.AssertNoUpdate(t, next, "on adding IP to non-existent link")

	msgCh <- msgDelAddrs(3, net.IPv4(192, 168, 1, 1), nil)
	notifier.AssertNoUpdate(t, next, "on removing IP from non-existent link")

	msgCh <- msgNewAddrs(1, net.IPv4(0, 0, 0, 0), nil)
	next = assertUpdated(t, next, sub)

	msgCh <- msgNewAddrs(1, net.IPv6loopback, nil)
	next = assertUpdated(t, next, sub)

	msgCh <- msgDelAddrs(1, net.IPv4(192, 168, 1, 1), net.IPv4(192, 168, 0, 1))
	next = assertUpdated(t, next, sub, "on removing an IP")
	eno1.IPs = []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback, net.IPv4(0, 0, 0, 0)}
	require.Equal(t, []Link{eno1}, sub.Get(), "All other link information is preserved")

	msgCh <- msgDelLink(3, Link{Name: "wlan0"})
	notifier.AssertNoUpdate(t, next, "on removing non-existent link")

	msgCh <- msgDelLink(1, Link{})
	assertUpdated(t, next, sub, "on deleting link")
	require.Empty(t, sub.Get(), "all links are gone")

	require.Empty(t, All().Get(), "when no links are present")
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
	nextW := subW.Next()

	subLocal := ByName("lo1")
	nextLocal := subLocal.Next()

	subEth := ByName("eth0")
	nextEth := subEth.Next()

	subAll := All()
	nextAll := subAll.Next()

	subAny := Any()
	nextAny := subAny.Next()

	notifier.AssertNoUpdate(t, nextW, "on start")
	notifier.AssertNoUpdate(t, nextLocal, "on start")
	notifier.AssertNoUpdate(t, nextEth, "on start")
	notifier.AssertNoUpdate(t, nextAll, "on start")
	notifier.AssertNoUpdate(t, nextAny, "on start")

	require.Equal(t, wlan0, subW.Get())
	require.Equal(t, lo1, subLocal.Get())
	require.Equal(t, Link{State: Gone}, subEth.Get())
	require.Equal(t, 4, len(subAll.Get()), "All four links in update to All()")
	require.Equal(t, wlan0, subAny.Get(), "Best link is sent to Any()")

	msgCh <- msgNewLink(1, Link{Name: "lo1", State: Up, HardwareAddr: hwA[4]})
	nextLocal = assertUpdated(t, nextLocal, subLocal, "Named link changed")
	notifier.AssertNoUpdate(t, nextEth, "Named link still not present")
	nextAll = assertUpdated(t, nextAll, subAll, "A link changed")
	nextAny = assertUpdated(t, nextAny, subAny, "A link changed")
	notifier.AssertNoUpdate(t, nextW, "No relevant link changed")

	msgCh <- msgNewLink(1, Link{Name: "lo1", State: Down, HardwareAddr: hwA[4]})
	nextLocal = assertUpdated(t, nextLocal, subLocal, "Named link changed")
	notifier.AssertNoUpdate(t, nextEth, "Named link still not present")
	nextAll = assertUpdated(t, nextAll, subAll, "A link changed")
	nextAny = assertUpdated(t, nextAny, subAny, "A link changed")
	notifier.AssertNoUpdate(t, nextW, "No relevant link changed")

	msgCh <- msgNewLink(4, Link{Name: "wwan0", State: Up, HardwareAddr: hwA[7]})
	nextW = assertUpdated(t, nextW, subW)
	notifier.AssertNoUpdate(t, nextLocal, "Named link unchanged")
	notifier.AssertNoUpdate(t, nextEth, "Named link still not present")
	nextAll = assertUpdated(t, nextAll, subAll, "A link changed")
	nextAny = assertUpdated(t, nextAny, subAny, "A link changed")
	require.Equal(t, wlan0, subAny.Get(), "'best' link is still the same")

	msgCh <- msgNewLink(2, Link{Name: "wlan0", State: Dormant, HardwareAddr: hwA[5]})
	assertUpdated(t, nextW, subW, "Relevant link changed")
	wwan0.State = Up
	require.Equal(t, wwan0, subW.Get(), "Re-order on state change")
	assertUpdated(t, nextAny, subAny, "A link changed")
	require.Equal(t, wwan0, subAny.Get(), "Re-order on state change")
}

func TestTestMode(t *testing.T) {
	nlt := TestMode()

	subEth := ByName("eth0")
	nextEth := subEth.Next()
	notifier.AssertNoUpdate(t, nextEth)
	subAll := All()
	nextAll := subAll.Next()
	notifier.AssertNoUpdate(t, nextAll)

	id := nlt.AddLink(Link{Name: "eno1"})
	nextAll = assertUpdated(t, nextAll, subAll)
	notifier.AssertNoUpdate(t, nextEth)

	nlt.AddIP(id, net.IPv4(10, 0, 0, 1))
	nextAll = assertUpdated(t, nextAll, subAll)
	notifier.AssertNoUpdate(t, nextEth)

	nlt.RemoveIP(id, net.IPv4(10, 0, 0, 1))
	nextAll = assertUpdated(t, nextAll, subAll)
	notifier.AssertNoUpdate(t, nextEth)

	nlt.RemoveLink(id)
	nextAll = assertUpdated(t, nextAll, subAll)
	notifier.AssertNoUpdate(t, nextEth)

	id = nlt.AddLink(Link{Name: "eth0", HardwareAddr: hwA[8]})
	nextAll = assertUpdated(t, nextAll, subAll)
	nextEth = assertUpdated(t, nextEth, subEth)

	nlt.AddIP(id, net.IPv4(10, 0, 0, 1))
	nlt.AddIP(id, net.IPv4(10, 0, 2, 1))
	nlt.AddIP(id, net.IPv6loopback)

	nextAll = assertUpdated(t, nextAll, subAll)
	nextEth = assertUpdated(t, nextEth, subEth)

	nlt.RemoveIP(id, net.IPv4(10, 0, 2, 1))
	expected := Link{
		Name:         "eth0",
		IPs:          []net.IP{net.IPv4(10, 0, 0, 1), net.IPv6loopback},
		HardwareAddr: hwA[8],
	}
	nextEth = assertUpdated(t, nextEth, subEth)
	require.Equal(t, expected, subEth.Get())
	nextAll = assertUpdated(t, nextAll, subAll)
	require.Equal(t, []Link{expected}, subAll.Get())

	nlt.UpdateLink(id, Link{State: Up})
	expected.State = Up
	nextEth = assertUpdated(t, nextEth, subEth)
	require.Equal(t, expected, subEth.Get())
	nextAll = assertUpdated(t, nextAll, subAll)

	subEth.Unsubscribe()
	nlt.UpdateLink(id, Link{State: Dormant})
	notifier.AssertNoUpdate(t, nextEth, "after unsubscribe")

	nlt.UpdateLink(id, Link{HardwareAddr: hwA[9]})
	expected.State = Dormant
	expected.HardwareAddr = hwA[9]
	require.Equal(t, expected, Any().Get(), "unset properties are retained")
}
