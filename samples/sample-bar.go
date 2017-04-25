// Copyright 2017 Google Inc.
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

// sample-bar demonstrates a sample i3bar built using barista.
package main

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/clock"
	"github.com/soumya92/barista/modules/cputemp"
	"github.com/soumya92/barista/modules/media"
	"github.com/soumya92/barista/modules/meminfo"
	"github.com/soumya92/barista/modules/netspeed"
	"github.com/soumya92/barista/modules/sysinfo"
	"github.com/soumya92/barista/modules/volume"
	"github.com/soumya92/barista/modules/weather"
	"github.com/soumya92/barista/outputs"
)

func truncate(in string, l int) string {
	if len([]rune(in)) <= l {
		return in
	}
	return string([]rune(in)[:l-1]) + "⋯"
}

func hms(d time.Duration) (h int, m int, s int) {
	h = int(d.Hours())
	m = int(d.Minutes()) % 60
	s = int(d.Seconds()) % 60
	return
}

func formatMediaTime(d time.Duration) string {
	h, m, s := hms(d)
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func mediaFormatFunc(m media.Info) *bar.Output {
	if m.PlaybackStatus == media.Stopped || m.PlaybackStatus == media.Disconnected {
		return nil
	}
	artist := truncate(m.Artist, 20)
	title := truncate(m.Title, 40-len(artist))
	if len(title) < 20 {
		artist = truncate(m.Artist, 40-len(title))
	}
	if m.PlaybackStatus != media.Playing {
		return outputs.Textf("%s - %s", title, artist)
	}
	return outputs.Textf(
		"%s/%s: %s - %s",
		formatMediaTime(m.Position()),
		formatMediaTime(m.Length),
		title,
		artist,
	)
}

func startTaskManager(e bar.Event) {
	if e.Button == bar.ButtonLeft {
		exec.Command("xfce4-taskmanager").Run()
	}
}

func main() {

	localtime := clock.New(clock.OutputFormat("Mon Jan 2 15:04"))
	localtime.OnClick(func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			exec.Command("gsimplecal").Run()
		}
	})

	apiKey, err := ioutil.ReadFile("/etc/config/owm.apikey")
	if err != nil {
		panic(err)
	}
	wthr := weather.New(
		weather.Zipcode{"94043", "US"},
		weather.APIKey(string(apiKey)),
		weather.OutputTemplate(outputs.TextTemplate(`{{.Description}}, {{.Temperature.C}}℃`)),
	)

	vol := volume.New(
		volume.OutputFunc(func(v volume.Volume) *bar.Output {
			if v.Mute {
				return outputs.Text("MUT")
			}
			return outputs.Textf("%02d%%", v.Pct())
		}),
	)

	loadAvg := sysinfo.New(
		sysinfo.OutputFunc("loadAvg", func(s sysinfo.Info) *bar.Output {
			out := outputs.Textf("%0.2f %0.2f", s.Loads[0], s.Loads[2])
			// Load averages are unusually high for a few minutes after boot.
			if s.Uptime < 10*time.Minute {
				// so don't add colours until 10 minutes after system start.
				return out
			}
			switch {
			case s.Loads[0] > 128, s.Loads[2] > 64:
				out.Urgent = true
			case s.Loads[0] > 64, s.Loads[2] > 32:
				out.Color = bar.Color("red")
			case s.Loads[0] > 32, s.Loads[2] > 16:
				out.Color = bar.Color("yellow")
			}
			return out
		}),
	).Get("loadAvg")
	loadAvg.OnClick(startTaskManager)

	freeMem := meminfo.New(
		meminfo.OutputFunc("freeMem", func(m meminfo.Info) *bar.Output {
			out := outputs.Text(m.Available().IEC())
			freeGigs := m.Available().In("GiB")
			switch {
			case freeGigs < 0.5:
				out.Urgent = true
			case freeGigs < 1:
				out.Color = bar.Color("red")
			case freeGigs < 2:
				out.Color = bar.Color("yellow")
			case freeGigs > 12:
				out.Color = bar.Color("green")
			}
			return out
		}),
	).Get("freeMem")
	freeMem.OnClick(startTaskManager)

	temp := cputemp.New(
		cputemp.RefreshInterval(2*time.Second),
		cputemp.OutputFunc(func(temp cputemp.Temperature) *bar.Output {
			celcius := temp.C()
			out := outputs.Textf("%2d℃", celcius)
			switch {
			case celcius > 90:
				out.Urgent = true
			case celcius > 70:
				out.Color = bar.Color("red")
			case celcius > 60:
				out.Color = bar.Color("yellow")
			}
			return out
		}),
	)

	netspeedTpl := `{{.Tx.SI | printf "%5s"}} {{.Rx.SI | printf "%5s"}}`
	net := netspeed.New(
		netspeed.Interface("em1"),
		netspeed.RefreshInterval(2*time.Second),
		netspeed.OutputTemplate(outputs.TextTemplate(netspeedTpl)),
	)

	rhythmbox := media.New(
		"rhythmbox",
		media.TrackPosition,
		media.OutputFunc(mediaFormatFunc),
	)

	panic(bar.Run(
		rhythmbox,
		net,
		temp,
		freeMem,
		loadAvg,
		vol,
		wthr,
		localtime,
	))
}
