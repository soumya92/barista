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

// i3status is a port of the default i3status configuration to barista.
package main

import (
	"fmt"
	"time"

	"barista.run"
	"barista.run/bar"
	"barista.run/colors"
	"barista.run/format"
	"barista.run/modules/battery"
	"barista.run/modules/clock"
	"barista.run/modules/diskspace"
	"barista.run/modules/meminfo"
	"barista.run/modules/netinfo"
	"barista.run/modules/sysinfo"
	"barista.run/modules/wlan"
	"barista.run/outputs"
	"github.com/martinlindhe/unit"
)

func main() {
	colors.LoadFromMap(map[string]string{
		"good":     "#0f0",
		"bad":      "#f00",
		"degraded": "#ff0",
	})

	barista.Add(netinfo.New().Output(func(s netinfo.State) bar.Output {
		if !s.Enabled() {
			return nil
		}
		for _, ip := range s.IPs {
			if ip.To4() == nil && ip.IsGlobalUnicast() {
				return outputs.Text(ip.String()).Color(colors.Scheme("good"))
			}
		}
		return outputs.Text("no IPv6").Color(colors.Scheme("bad"))
	}))

	barista.Add(diskspace.New("/").Output(func(i diskspace.Info) bar.Output {
		out := outputs.Text(format.IBytesize(i.Available))
		switch {
		case i.AvailFrac() < 0.2:
			out.Color(colors.Scheme("bad"))
		case i.AvailFrac() < 0.33:
			out.Color(colors.Scheme("degraded"))
		}
		return out
	}))

	barista.Add(wlan.Any().Output(func(w wlan.Info) bar.Output {
		switch {
		case w.Connected():
			out := fmt.Sprintf("W: (%s)", w.SSID)
			if len(w.IPs) > 0 {
				out += fmt.Sprintf(" %s", w.IPs[0])
			}
			return outputs.Text(out).Color(colors.Scheme("good"))
		case w.Connecting():
			return outputs.Text("W: connecting...").Color(colors.Scheme("degraded"))
		case w.Enabled():
			return outputs.Text("W: down").Color(colors.Scheme("bad"))
		default:
			return nil
		}
	}))

	barista.Add(netinfo.Prefix("e").Output(func(s netinfo.State) bar.Output {
		switch {
		case s.Connected():
			ip := "<no ip>"
			if len(s.IPs) > 0 {
				ip = s.IPs[0].String()
			}
			return outputs.Textf("E: %s", ip).Color(colors.Scheme("good"))
		case s.Connecting():
			return outputs.Text("E: connecting...").Color(colors.Scheme("degraded"))
		case s.Enabled():
			return outputs.Text("E: down").Color(colors.Scheme("bad"))
		default:
			return nil
		}
	}))

	statusName := map[battery.Status]string{
		battery.Charging:    "CHR",
		battery.Discharging: "BAT",
		battery.NotCharging: "NOT",
		battery.Unknown:     "UNK",
	}
	barista.Add(battery.All().Output(func(b battery.Info) bar.Output {
		if b.Status == battery.Disconnected {
			return nil
		}
		if b.Status == battery.Full {
			return outputs.Text("FULL")
		}
		out := outputs.Textf("%s %d%% %s",
			statusName[b.Status],
			b.RemainingPct(),
			b.RemainingTime())
		if b.Discharging() {
			if b.RemainingPct() < 20 || b.RemainingTime() < 30*time.Minute {
				out.Color(colors.Scheme("bad"))
			}
		}
		return out
	}))

	barista.Add(sysinfo.New().Output(func(i sysinfo.Info) bar.Output {
		out := outputs.Textf("%.2f", i.Loads[0])
		if i.Loads[0] > 5.0 {
			out.Color(colors.Scheme("bad"))
		}
		return out
	}))

	barista.Add(meminfo.New().Output(func(i meminfo.Info) bar.Output {
		if i.Available() < unit.Gigabyte {
			return outputs.Textf(`MEMORY < %s`,
				format.IBytesize(i.Available())).
				Color(colors.Scheme("bad"))
		}
		out := outputs.Textf(`%s/%s`,
			format.IBytesize(i["MemTotal"]-i.Available()),
			format.IBytesize(i.Available()))
		switch {
		case i.AvailFrac() < 0.2:
			out.Color(colors.Scheme("bad"))
		case i.AvailFrac() < 0.33:
			out.Color(colors.Scheme("degraded"))
		}
		return out
	}))

	barista.Add(clock.Local().OutputFormat("2006-01-02 15:04:05"))

	panic(barista.Run())
}
