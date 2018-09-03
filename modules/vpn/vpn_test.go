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

	"barista.run/bar"
	"barista.run/base/watchers/netlink"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
)

func TestVpn(t *testing.T) {
	nlt := netlink.TestMode()
	link := nlt.AddLink(netlink.Link{Name: "tun0", State: netlink.Down})

	testBar.New(t)
	v := DefaultInterface()

	testBar.Run(v)
	testBar.NextOutput().AssertText([]string{})

	nlt.UpdateLink(link, netlink.Link{Name: "tun0", State: netlink.Up})
	testBar.NextOutput().AssertText([]string{"VPN"})

	v.Output(func(s State) bar.Output {
		switch {
		case s.Connected():
			return outputs.Text("VPN!")
		case s.Disconnected():
			return nil
		default:
			return outputs.Text("...")
		}
	})
	testBar.NextOutput().AssertText([]string{"VPN!"})

	nlt.UpdateLink(link, netlink.Link{Name: "tun0", State: netlink.Dormant})
	testBar.NextOutput().AssertText([]string{"..."})

	nlt.UpdateLink(link, netlink.Link{Name: "tun0", State: netlink.Up})
	testBar.NextOutput().AssertText([]string{"VPN!"})

	v.Output(func(s State) bar.Output {
		if s.Disconnected() {
			return outputs.Text("NO VPN")
		}
		return nil
	})
	testBar.NextOutput().AssertText([]string{})

	nlt.RemoveLink(link)
	testBar.NextOutput().AssertText([]string{"NO VPN"})
}
