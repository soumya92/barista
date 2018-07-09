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

package vpn

import (
	"testing"

	"github.com/soumya92/barista/base/watchers/netlink"
	testBar "github.com/soumya92/barista/testing/bar"
)

func TestVpn(t *testing.T) {
	nlt := netlink.TestMode()
	link := nlt.AddLink(netlink.Link{Name: "tun0", State: netlink.Down})

	testBar.New(t)
	v := DefaultInterface().Template(
		`{{if not .Disconnected}}{{if .Connected}}VPN{{else}}...{{end}}{{end}}`)
	testBar.Run(v)

	testBar.LatestOutput().AssertText([]string{""})

	nlt.UpdateLink(link, netlink.Link{Name: "tun0", State: netlink.Dormant})
	testBar.LatestOutput().AssertText([]string{"..."})

	nlt.UpdateLink(link, netlink.Link{Name: "tun0", State: netlink.Up})
	testBar.LatestOutput().AssertText([]string{"VPN"})

	v.Template(`{{if .Disconnected}}NO VPN{{end}}`)
	testBar.LatestOutput().AssertText([]string{""})

	nlt.RemoveLink(link)
	testBar.LatestOutput().AssertText([]string{"NO VPN"})
}
