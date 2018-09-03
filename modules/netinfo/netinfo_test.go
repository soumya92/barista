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

package netinfo

import (
	"testing"

	"barista.run/bar"
	"barista.run/base/watchers/netlink"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
)

func TestNetinfo(t *testing.T) {
	nlt := netlink.TestMode()
	link0 := nlt.AddLink(netlink.Link{Name: "lo0", State: netlink.Up})
	link1 := nlt.AddLink(netlink.Link{Name: "eth0", State: netlink.Down})
	link2 := nlt.AddLink(netlink.Link{Name: "eth1", State: netlink.Down})
	link3 := nlt.AddLink(netlink.Link{Name: "wlan0", State: netlink.NotPresent})

	testBar.New(t)
	n1 := New().Output(func(s State) bar.Output {
		if !s.Connected() {
			return outputs.Text("No net")
		}
		return outputs.Text(s.Name)
	})
	n2 := Interface("wlan0").Output(func(s State) bar.Output {
		if !s.Enabled() {
			return nil
		}
		if s.Connected() {
			return outputs.Textf("W:%s", s.Name)
		} else if s.Connecting() {
			return outputs.Text("W:...")
		} else {
			return outputs.Text("W:down")
		}
	})
	n3 := Prefix("eth").Output(func(s State) bar.Output {
		if !s.Connected() {
			return nil
		}
		return outputs.Textf("E:%s", s.Name)
	})
	n4 := New()
	testBar.Run(n1, n2, n3, n4)

	testBar.LatestOutput().AssertText([]string{"lo0", "lo0"})

	nlt.UpdateLink(link0, netlink.Link{State: netlink.Down})
	testBar.LatestOutput(0, 3).Expect("on link update")
	nlt.UpdateLink(link3, netlink.Link{State: netlink.Down})
	testBar.LatestOutput(0, 1, 3).AssertText([]string{"No net", "W:down"})

	nlt.UpdateLink(link1, netlink.Link{State: netlink.Dormant})
	testBar.LatestOutput(0, 2, 3).Expect("on link update")
	nlt.UpdateLink(link2, netlink.Link{State: netlink.Up})
	testBar.LatestOutput(0, 2, 3).Expect("on link update")

	n1.Output(func(s State) bar.Output {
		return outputs.Textf("%v", s.State)
	})
	testBar.NextOutput().AssertText([]string{"6", "W:down", "E:eth1", "eth1"})
}
