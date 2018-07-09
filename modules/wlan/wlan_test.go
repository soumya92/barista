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

	"github.com/soumya92/barista/base/watchers/netlink"
	testBar "github.com/soumya92/barista/testing/bar"
)

func TestNoWlan(t *testing.T) {
	netlink.TestMode()
	testBar.New(t)
	wl := New("wlan0")
	testBar.Run(wl)
	testBar.AssertNoOutput("when wlan link is missing")
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
	wl0 := New("wlan0").Template(
		`{{if .Connected}}{{.Frequency | printf "%.1g"}}{{end}}`)
	wl1 := New("wlan1").Template(
		`{{if .Enabled -}}
			{{- if .Connecting -}}
				WLAN ...
			{{- else if .Connected -}}
				{{.SSID}}
			{{- else -}}
				WL: Down
			{{- end -}}
		{{- end}}`)
	testBar.Run(wl0, wl1)

	testBar.LatestOutput().AssertText([]string{"5e+09", "WLAN ..."})

	iwgetidShouldReturn("wlan1", map[string]string{
		"-r": "NetworkName",
		"-a": "00:11:22:33:44:55",
		"-c": "11",
		"-f": "2.4e+09",
	})
	nlt.UpdateLink(link1, netlink.Link{Name: "wlan1", State: netlink.Up})
	testBar.LatestOutput().At(1).AssertText("NetworkName")

	wl0.Template(`{{index .IPs 0}}`)
	nlt.AddIP(link0, net.IPv4(10, 0, 0, 1))
	testBar.LatestOutput().At(0).AssertText("10.0.0.1")

	nlt.AddIP(link0, net.IPv6loopback)
	wl0.Template(`{{index .IPs 1}}`)
	testBar.LatestOutput().At(0).AssertText("::1")
}
