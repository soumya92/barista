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

package wlan

import (
	"errors"
	"net"
	"sync"
	"testing"

	"barista.run/bar"
	"barista.run/base/watchers/netlink"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
)

func TestNoWlan(t *testing.T) {
	netlink.TestMode()
	testBar.New(t)
	wlN := Named("wlan0")
	wlA := Any()
	testBar.Run(wlN, wlA)
	testBar.LatestOutput().AssertEmpty("when no link is present")
}

// Map of interface -> map of iwgetid flag -> value.
var (
	testData = map[string]map[string]string{}
	testMu   sync.RWMutex
)

func mockIwgetid(intf, flag string) (string, error) {
	testMu.RLock()
	defer testMu.RUnlock()
	d, ok := testData[intf]
	if !ok {
		return "", errors.New("No interface")
	}
	val, ok := d[flag]
	if !ok {
		return "", errors.New("Unknown flag")
	}
	return val, nil
}

func iwgetidShouldReturn(intf string, data map[string]string) {
	testMu.Lock()
	defer testMu.Unlock()
	testData[intf] = data
}

func init() {
	iwgetid = mockIwgetid
}

func TestWlan(t *testing.T) {
	nlt := netlink.TestMode()
	iwgetidShouldReturn("wlan0", map[string]string{
		"-r": "OtherNet",
		"-a": "00:11:22:33:44:66",
		"-c": "141",
		"-f": "5.22e+09",
	})
	link0 := nlt.AddLink(netlink.Link{Name: "wlan0", State: netlink.Up})
	link1 := nlt.AddLink(netlink.Link{Name: "wlan1", State: netlink.Dormant})

	testBar.New(t)
	wl0 := Named("wlan0")
	wl1 := Named("wlan1")
	wlA := Any()
	testBar.Run(wl0, wl1, wlA)
	testBar.LatestOutput().Expect("on start")

	wl0.Output(func(i Info) bar.Output {
		if i.Connected() {
			return outputs.Textf("%.1g", i.Frequency)
		}
		return nil
	})
	wl1.Output(func(i Info) bar.Output {
		if !i.Enabled() {
			return nil
		}
		switch {
		case i.Connecting():
			return outputs.Text("WLAN ...")
		case i.Connected():
			return outputs.Text(i.SSID)
		default:
			return outputs.Text("WL: Down")
		}
	})
	wlA.Output(func(i Info) bar.Output {
		if i.Enabled() {
			return outputs.Textf("%s/%s", i.Name, i.SSID)
		}
		return outputs.Text("<no wlan>")
	})
	testBar.LatestOutput().AssertText([]string{"5e+09", "WLAN ...", "wlan0/OtherNet"})

	iwgetidShouldReturn("wlan1", map[string]string{
		"-r": "NetworkName",
		"-a": "00:11:22:33:44:55",
		"-c": "11",
		"-f": "2.4e+09",
	})
	nlt.UpdateLink(link1, netlink.Link{Name: "wlan1", State: netlink.Up})
	testBar.LatestOutput(1, 2).AssertText([]string{"5e+09", "NetworkName", "wlan0/OtherNet"})

	wl0.Output(func(i Info) bar.Output {
		if len(i.IPs) > 0 {
			return outputs.Textf("%v", i.IPs[0])
		}
		return nil
	})
	testBar.LatestOutput(0).Expect("on template change")
	nlt.AddIP(link0, net.IPv4(10, 0, 0, 1))
	testBar.LatestOutput(0, 2).At(0).AssertText("10.0.0.1")

	wl0.Output(func(i Info) bar.Output {
		if len(i.IPs) > 1 {
			return outputs.Textf("%v", i.IPs[1])
		}
		return nil
	})
	testBar.LatestOutput(0).Expect("on template change")
	nlt.AddIP(link0, net.IPv6loopback)
	testBar.LatestOutput(0, 2).At(0).AssertText("::1")

	nlt.UpdateLink(link0, netlink.Link{Name: "wlan0", State: netlink.Down})
	testBar.LatestOutput(0, 2).At(2).AssertText("wlan1/NetworkName", "when active link switches")

	iwgetidShouldReturn("wl1", map[string]string{
		"-r": "NetworkName",
		"-a": "00:11:22:33:44:55",
		"-c": "11",
		"-f": "2.4e+09",
	})
	nlt.UpdateLink(link1, netlink.Link{Name: "wl1", State: netlink.Up})
	testBar.LatestOutput(1, 2).AssertText([]string{"::1", "wl1/NetworkName"}, "when active link is renamed")

	nlt.RemoveLink(link1)
	testBar.LatestOutput(2).AssertText([]string{"::1", "wlan0/OtherNet"}, "fallback when active link is removed")

	nlt.RemoveLink(link0)
	testBar.LatestOutput(0, 2).AssertText([]string{"<no wlan>"}, "when no links remain")
}
