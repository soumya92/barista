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

	"github.com/soumya92/barista/base/watchers/netlink"
	testBar "github.com/soumya92/barista/testing/bar"
)

func TestNetinfo(t *testing.T) {
	nlt := netlink.TestMode()
	link0 := nlt.AddLink(netlink.Link{Name: "lo0", State: netlink.Up})
	link1 := nlt.AddLink(netlink.Link{Name: "eth0", State: netlink.Down})
	link2 := nlt.AddLink(netlink.Link{Name: "eth1", State: netlink.Down})
	link3 := nlt.AddLink(netlink.Link{Name: "wlan0", State: netlink.NotPresent})

	testBar.New(t)
	n1 := New().Template(`{{if not .Connected}}No net{{else}}{{.Name}}{{end}}`)
	n2 := Interface("wlan0").Template(`{{if .Enabled}}W:{{if .Connected}}{{.Name}}{{else}}down{{end}}{{end}}`)
	n3 := Prefix("eth").Template(`{{if .Connected}}E:{{.Name}}{{end}}`)
	testBar.Run(n1, n2, n3)

	testBar.LatestOutput().AssertText([]string{"lo0", "", ""})

	nlt.UpdateLink(link0, netlink.Link{State: netlink.Down})
	nlt.UpdateLink(link3, netlink.Link{State: netlink.Down})
	testBar.LatestOutput().AssertText([]string{"No net", "W:down", ""})

	nlt.UpdateLink(link1, netlink.Link{State: netlink.Dormant})
	nlt.UpdateLink(link2, netlink.Link{State: netlink.Up})
	n1.Template(`{{.State}}`)
	testBar.LatestOutput().AssertText([]string{"6", "W:down", "E:eth1"})
}
