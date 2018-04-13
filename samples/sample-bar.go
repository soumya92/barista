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
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/modules/clock"
	"github.com/soumya92/barista/modules/cputemp"
	"github.com/soumya92/barista/modules/group"
	"github.com/soumya92/barista/modules/media"
	"github.com/soumya92/barista/modules/meminfo"
	"github.com/soumya92/barista/modules/netspeed"
	"github.com/soumya92/barista/modules/sysinfo"
	"github.com/soumya92/barista/modules/volume"
	"github.com/soumya92/barista/modules/weather"
	"github.com/soumya92/barista/modules/weather/openweathermap"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/pango/icons/fontawesome"
	"github.com/soumya92/barista/pango/icons/ionicons"
	"github.com/soumya92/barista/pango/icons/material"
	"github.com/soumya92/barista/pango/icons/material_community"
	"github.com/soumya92/barista/pango/icons/typicons"
)

var spacer = pango.Span(" ", pango.XXSmall)

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

func mediaFormatFunc(m media.Info) bar.Output {
	if m.PlaybackStatus == media.Stopped || m.PlaybackStatus == media.Disconnected {
		return outputs.Empty()
	}
	artist := truncate(m.Artist, 20)
	title := truncate(m.Title, 40-len(artist))
	if len(title) < 20 {
		artist = truncate(m.Artist, 40-len(title))
	}
	var iconAndPosition pango.Node
	if m.PlaybackStatus == media.Playing {
		iconAndPosition = pango.Span(
			colors.Hex("#f70"),
			fontawesome.Icon("music"),
			spacer,
			formatMediaTime(m.Position()),
			"/",
			formatMediaTime(m.Length),
		)
	} else {
		iconAndPosition = fontawesome.Icon("music", colors.Hex("#f70"))
	}
	return outputs.Pango(iconAndPosition, spacer, title, " - ", artist)
}

func startTaskManager(e bar.Event) {
	if e.Button == bar.ButtonLeft {
		exec.Command("xfce4-taskmanager").Run()
	}
}

func home(path string) string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	return filepath.Join(usr.HomeDir, path)
}

func main() {
	material.Load(home("Github/material-design-icons"))
	materialCommunity.Load(home("Github/MaterialDesign-Webfont"))
	typicons.Load(home("Github/typicons.font"))
	ionicons.Load(home("Github/ionicons"))
	fontawesome.Load(home("Github/Font-Awesome"))

	colors.LoadFromMap(map[string]string{
		"good":     "#6d6",
		"degraded": "#dd6",
		"bad":      "#d66",
		"dim-icon": "#777",
	})

	localtime := clock.New().OutputFunc(func(now time.Time) bar.Output {
		return outputs.Pango(
			material.Icon("today", colors.Scheme("dim-icon")),
			now.Format("Mon Jan 2 "),
			material.Icon("access-time", colors.Scheme("dim-icon")),
			now.Format("15:04:05"),
		)
	}).OnClick(func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			exec.Command("gsimplecal").Run()
		}
	})

	// Weather information comes from OpenWeatherMap.
	// https://openweathermap.org/api.
	wthr := weather.New(
		openweathermap.Zipcode("94043", "US").Build(),
	).OutputFunc(func(w weather.Weather) bar.Output {
		iconName := ""
		switch w.Condition {
		case weather.Thunderstorm,
			weather.TropicalStorm,
			weather.Hurricane:
			iconName = "stormy"
		case weather.Drizzle,
			weather.Hail:
			iconName = "shower"
		case weather.Rain:
			iconName = "downpour"
		case weather.Snow,
			weather.Sleet:
			iconName = "snow"
		case weather.Mist,
			weather.Smoke,
			weather.Whirls,
			weather.Haze,
			weather.Fog:
			iconName = "windy-cloudy"
		case weather.Clear:
			if !w.Sunset.IsZero() && time.Now().After(w.Sunset) {
				iconName = "night"
			} else {
				iconName = "sunny"
			}
		case weather.PartlyCloudy:
			iconName = "partly-sunny"
		case weather.Cloudy, weather.Overcast:
			iconName = "cloudy"
		case weather.Tornado,
			weather.Windy:
			iconName = "windy"
		}
		if iconName == "" {
			iconName = "warning-outline"
		} else {
			iconName = "weather-" + iconName
		}
		return outputs.Pango(
			typicons.Icon(iconName), spacer,
			pango.Textf("%d℃", w.Temperature.C()),
			pango.Span(" (provided by ", w.Attribution, ")", pango.XSmall),
		)
	})

	vol := volume.DefaultMixer().OutputFunc(func(v volume.Volume) bar.Output {
		if v.Mute {
			return outputs.
				Pango(ionicons.Icon("volume-mute"), "MUT").
				Color(colors.Scheme("degraded"))
		}
		iconName := "low"
		pct := v.Pct()
		if pct > 66 {
			iconName = "high"
		} else if pct > 33 {
			iconName = "medium"
		}
		return outputs.Pango(
			ionicons.Icon("volume-"+iconName),
			spacer,
			pango.Textf("%2d%%", pct),
		)
	})

	loadAvg := sysinfo.New().OutputFunc(func(s sysinfo.Info) bar.Output {
		out := outputs.Textf("%0.2f %0.2f", s.Loads[0], s.Loads[2])
		// Load averages are unusually high for a few minutes after boot.
		if s.Uptime < 10*time.Minute {
			// so don't add colours until 10 minutes after system start.
			return out
		}
		switch {
		case s.Loads[0] > 128, s.Loads[2] > 64:
			out.Urgent(true)
		case s.Loads[0] > 64, s.Loads[2] > 32:
			out.Color(colors.Scheme("bad"))
		case s.Loads[0] > 32, s.Loads[2] > 16:
			out.Color(colors.Scheme("degraded"))
		}
		return out
	}).OnClick(startTaskManager)

	freeMem := meminfo.New().OutputFunc(func(m meminfo.Info) bar.Output {
		out := outputs.Pango(material.Icon("memory"), m.Available().IEC())
		freeGigs := m.Available().In("GiB")
		switch {
		case freeGigs < 0.5:
			out.Urgent(true)
		case freeGigs < 1:
			out.Color(colors.Scheme("bad"))
		case freeGigs < 2:
			out.Color(colors.Scheme("degraded"))
		case freeGigs > 12:
			out.Color(colors.Scheme("good"))
		}
		return out
	}).OnClick(startTaskManager)

	temp := cputemp.DefaultZone().
		RefreshInterval(2 * time.Second).
		UrgentWhen(func(temp cputemp.Temperature) bool {
			return temp.C() > 90
		}).
		OutputColor(func(temp cputemp.Temperature) bar.Color {
			switch {
			case temp.C() > 70:
				return colors.Scheme("bad")
			case temp.C() > 60:
				return colors.Scheme("degraded")
			default:
				return colors.Empty()
			}
		}).
		OutputFunc(func(temp cputemp.Temperature) bar.Output {
			return outputs.Pango(
				materialCommunity.Icon("fan"), spacer,
				pango.Textf("%2d℃", temp.C()),
			)
		})

	net := netspeed.New("eno1").
		RefreshInterval(2 * time.Second).
		OutputFunc(func(s netspeed.Speeds) bar.Output {
			return outputs.Pango(
				fontawesome.Icon("upload"), spacer, pango.Textf("%5s", s.Tx.SI()),
				pango.Span(" ", pango.Small),
				fontawesome.Icon("download"), spacer, pango.Textf("%5s", s.Rx.SI()),
			)
		})

	rhythmbox := media.New("rhythmbox").OutputFunc(mediaFormatFunc)

	g := group.Collapsing()

	panic(bar.Run(
		rhythmbox,
		g.Add(net),
		g.Add(temp),
		g.Add(freeMem),
		g.Add(loadAvg),
		g.Button(outputs.Text("+"), outputs.Text("-")),
		vol,
		wthr,
		localtime,
	))
}
