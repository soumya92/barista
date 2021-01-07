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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"barista.run"
	"barista.run/bar"
	"barista.run/base/click"
	"barista.run/base/watchers/netlink"
	"barista.run/colors"
	"barista.run/format"
	"barista.run/group/modal"
	"barista.run/modules/battery"
	"barista.run/modules/clock"
	"barista.run/modules/cputemp"
	"barista.run/modules/diskio"
	"barista.run/modules/diskspace"
	"barista.run/modules/github"
	"barista.run/modules/media"
	"barista.run/modules/meminfo"
	"barista.run/modules/meta/split"
	"barista.run/modules/netinfo"
	"barista.run/modules/netspeed"
	"barista.run/modules/sysinfo"
	"barista.run/modules/volume"
	"barista.run/modules/volume/alsa"
	"barista.run/modules/weather"
	"barista.run/modules/weather/openweathermap"
	"barista.run/modules/wlan"
	"barista.run/oauth"
	"barista.run/outputs"
	"barista.run/pango"
	"barista.run/pango/icons/fontawesome"
	"barista.run/pango/icons/material"
	"barista.run/pango/icons/mdi"
	"barista.run/pango/icons/typicons"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/martinlindhe/unit"
	keyring "github.com/zalando/go-keyring"
)

var spacer = pango.Text(" ").XXSmall()
var mainModalController modal.Controller

func truncate(in string, l int) string {
	fromStart := false
	if l < 0 {
		fromStart = true
		l = -l
	}
	inLen := len([]rune(in))
	if inLen <= l {
		return in
	}
	if fromStart {
		return "⋯" + string([]rune(in)[inLen-l+1:])
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

func makeMediaIconAndPosition(m media.Info) *pango.Node {
	iconAndPosition := pango.Icon("fa-music").Color(colors.Hex("#f70"))
	if m.PlaybackStatus == media.Playing {
		iconAndPosition.Append(spacer,
			pango.Textf("%s/", formatMediaTime(m.Position())))
	}
	if m.PlaybackStatus == media.Paused || m.PlaybackStatus == media.Playing {
		iconAndPosition.Append(spacer,
			pango.Textf("%s", formatMediaTime(m.Length)))
	}
	return iconAndPosition
}

func mediaFormatFunc(m media.Info) bar.Output {
	if m.PlaybackStatus == media.Stopped || m.PlaybackStatus == media.Disconnected {
		return nil
	}
	artist := truncate(m.Artist, 35)
	title := truncate(m.Title, 70-len(artist))
	if len(title) < 35 {
		artist = truncate(m.Artist, 35-len(title))
	}
	var iconAndPosition bar.Output
	if m.PlaybackStatus == media.Playing {
		iconAndPosition = outputs.Repeat(func(time.Time) bar.Output {
			return makeMediaIconAndPosition(m)
		}).Every(time.Second)
	} else {
		iconAndPosition = makeMediaIconAndPosition(m)
	}
	return outputs.Group(iconAndPosition, outputs.Pango(title, " - ", artist))
}

func home(path ...string) string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	args := append([]string{usr.HomeDir}, path...)
	return filepath.Join(args...)
}

func deviceForMountPath(path string) string {
	mnt, _ := exec.Command("df", "-P", path).Output()
	lines := strings.Split(string(mnt), "\n")
	if len(lines) > 1 {
		devAlias := strings.Split(lines[1], " ")[0]
		dev, _ := exec.Command("realpath", devAlias).Output()
		devStr := strings.TrimSpace(string(dev))
		if devStr != "" {
			return devStr
		}
		return devAlias
	}
	return ""
}

type freegeoipResponse struct {
	Lat float64 `json:"latitude"`
	Lng float64 `json:"longitude"`
}

func whereami() (lat float64, lng float64, err error) {
	resp, err := http.Get("https://freegeoip.app/json/")
	if err != nil {
		return 0, 0, err
	}
	var res freegeoipResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return 0, 0, err
	}
	return res.Lat, res.Lng, nil
}

type autoWeatherProvider struct{}

func (a autoWeatherProvider) GetWeather() (weather.Weather, error) {
	lat, lng, err := whereami()
	if err != nil {
		return weather.Weather{}, err
	}
	return openweathermap.
		New("%%OWM_API_KEY%%").
		Coords(lat, lng).
		GetWeather()
}

func setupOauthEncryption() error {
	const service = "barista-sample-bar"
	var username string
	if u, err := user.Current(); err == nil {
		username = u.Username
	} else {
		username = fmt.Sprintf("user-%d", os.Getuid())
	}
	var secretBytes []byte
	// IMPORTANT: The oauth tokens used by some modules are very sensitive, so
	// we encrypt them with a random key and store that random key using
	// libsecret (gnome-keyring or equivalent). If no secret provider is
	// available, there is no way to store tokens (since the version of
	// sample-bar used for setup-oauth will have a different key from the one
	// running in i3bar). See also https://github.com/zalando/go-keyring#linux.
	secret, err := keyring.Get(service, username)
	if err == nil {
		secretBytes, err = base64.RawURLEncoding.DecodeString(secret)
	}
	if err != nil {
		secretBytes = make([]byte, 64)
		_, err := rand.Read(secretBytes)
		if err != nil {
			return err
		}
		secret = base64.RawURLEncoding.EncodeToString(secretBytes)
		err = keyring.Set(service, username, secret)
		if err != nil {
			return err
		}
	}
	oauth.SetEncryptionKey(secretBytes)
	return nil
}

func makeIconOutput(key string) *bar.Segment {
	return outputs.Pango(spacer, pango.Icon(key), spacer)
}

var gsuiteOauthConfig = []byte(`{"installed": {
	"client_id":"%%GOOGLE_CLIENT_ID%%",
	"project_id":"i3-barista",
	"auth_uri":"https://accounts.google.com/o/oauth2/auth",
	"token_uri":"https://www.googleapis.com/oauth2/v3/token",
	"auth_provider_x509_cert_url":"https://www.googleapis.com/oauth2/v1/certs",
	"client_secret":"%%GOOGLE_CLIENT_SECRET%%",
	"redirect_uris":["urn:ietf:wg:oauth:2.0:oob","http://localhost"]
}}`)

func threshold(out *bar.Segment, urgent bool, color ...bool) *bar.Segment {
	if urgent {
		return out.Urgent(true)
	}
	colorKeys := []string{"bad", "degraded", "good"}
	for i, c := range colorKeys {
		if len(color) > i && color[i] {
			return out.Color(colors.Scheme(c))
		}
	}
	return out
}

func main() {
	material.Load(home("Github/material-design-icons"))
	mdi.Load(home("Github/MaterialDesign-Webfont"))
	typicons.Load(home("Github/typicons.font"))
	fontawesome.Load(home("Github/Font-Awesome"))

	colors.LoadBarConfig()
	bg := colors.Scheme("background")
	fg := colors.Scheme("statusline")
	if fg != nil && bg != nil {
		_, _, v := fg.Colorful().Hsv()
		if v < 0.3 {
			v = 0.3
		}
		colors.Set("bad", colorful.Hcl(40, 1.0, v).Clamped())
		colors.Set("degraded", colorful.Hcl(90, 1.0, v).Clamped())
		colors.Set("good", colorful.Hcl(120, 1.0, v).Clamped())
	}

	if err := setupOauthEncryption(); err != nil {
		panic(fmt.Sprintf("Could not setup oauth token encryption: %v", err))
	}

	localdate := clock.Local().
		Output(time.Second, func(now time.Time) bar.Output {
			return outputs.Pango(
				pango.Icon("material-today").Alpha(0.6),
				now.Format("Mon Jan 2"),
			).OnClick(click.RunLeft("gsimplecal"))
		})

	localtime := clock.Local().
		Output(time.Second, func(now time.Time) bar.Output {
			return outputs.Text(now.Format("15:04:05")).
				OnClick(click.Left(func() {
					mainModalController.Toggle("timezones")
				}))
		})

	makeTzClock := func(lbl, tzName string) bar.Module {
		c, err := clock.ZoneByName(tzName)
		if err != nil {
			panic(err)
		}
		return c.Output(time.Minute, func(now time.Time) bar.Output {
			return outputs.Pango(pango.Text(lbl).Smaller(), spacer, now.Format("15:04"))
		})
	}

	// Weather information comes from OpenWeatherMap.
	// https://openweathermap.org/api.
	wthr := weather.New(autoWeatherProvider{}).Output(func(w weather.Weather) bar.Output {
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
			} else if !w.Sunrise.IsZero() && time.Now().Before(w.Sunrise) {
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
		mainModalController.SetOutput("weather", makeIconOutput("typecn-"+iconName))
		out := outputs.Group()
		out.Append(outputs.Pango(
			pango.Icon("typecn-"+iconName), spacer,
			pango.Textf("%.1f℃", w.Temperature.Celsius()),
		))
		out.Append(outputs.Text(w.Description))
		out.Append(outputs.Pango(
			pango.Icon("mdi-flag-variant-outline").Alpha(0.8), spacer,
			pango.Textf("%0.fmph %s", w.Wind.Speed.MilesPerHour(), w.Wind.Direction.Cardinal()),
		))
		out.Append(outputs.Pango(
			pango.Icon("fa-tint").Alpha(0.6).Small(), spacer,
			pango.Textf("%0.f%%", w.Humidity*100),
		))
		out.Append(outputs.Pango(
			pango.Icon("mdi-weather-sunset-up").Alpha(0.8), spacer,
			w.Sunrise.Format("15:04"), spacer,
			pango.Icon("mdi-weather-sunset-down").Alpha(0.8), spacer,
			w.Sunset.Format("15:04"),
		))
		out.Append(pango.Textf("provided by %s", w.Attribution).XSmall())
		return out
	})

	battSummary, battDetail := split.New(battery.All().Output(func(i battery.Info) bar.Output {
		if i.Status == battery.Disconnected || i.Status == battery.Unknown {
			return nil
		}
		iconName := "battery"
		if i.Status == battery.Charging {
			iconName += "-charging"
		}
		tenth := i.RemainingPct() / 10
		switch {
		case tenth == 0:
			iconName += "-outline"
		case tenth < 10:
			iconName += fmt.Sprintf("-%d0", tenth)
		}
		mainModalController.SetOutput("battery", makeIconOutput("mdi-"+iconName))
		rem := i.RemainingTime()
		out := outputs.Group()
		// First segment will be used in summary mode.
		out.Append(outputs.Pango(
			pango.Icon("mdi-"+iconName).Alpha(0.6),
			pango.Textf("%d:%02d", int(rem.Hours()), int(rem.Minutes())%60),
		).OnClick(click.Left(func() {
			mainModalController.Toggle("battery")
		})))
		// Others in detail mode.
		out.Append(outputs.Pango(
			pango.Icon("mdi-"+iconName).Alpha(0.6),
			pango.Textf("%d%%", i.RemainingPct()),
			spacer,
			pango.Textf("(%d:%02d)", int(rem.Hours()), int(rem.Minutes())%60),
		).OnClick(click.Left(func() {
			mainModalController.Toggle("battery")
		})))
		out.Append(outputs.Pango(
			pango.Textf("%4.1f/%4.1f", i.EnergyNow, i.EnergyFull),
			pango.Text("Wh").Smaller(),
		))
		out.Append(outputs.Pango(
			pango.Textf("% +6.2f", i.SignedPower()),
			pango.Text("W").Smaller(),
		))
		switch {
		case i.RemainingPct() <= 5:
			out.Urgent(true)
		case i.RemainingPct() <= 15:
			out.Color(colors.Scheme("bad"))
		case i.RemainingPct() <= 25:
			out.Color(colors.Scheme("degraded"))
		}
		return out
	}), 1)

	wifiName, wifiDetails := split.New(wlan.Any().Output(func(i wlan.Info) bar.Output {
		if !i.Connecting() && !i.Connected() {
			mainModalController.SetOutput("network", makeIconOutput("mdi-ethernet"))
			return nil
		}
		mainModalController.SetOutput("network", makeIconOutput("mdi-wifi"))
		if i.Connecting() {
			return outputs.Pango(pango.Icon("mdi-wifi").Alpha(0.6), "...").
				Color(colors.Scheme("degraded"))
		}
		out := outputs.Group()
		// First segment shown in summary mode only.
		out.Append(outputs.Pango(
			pango.Icon("mdi-wifi").Alpha(0.6),
			pango.Text(truncate(i.SSID, -9)),
		).OnClick(click.Left(func() {
			mainModalController.Toggle("network")
		})))
		// Full name, frequency, bssid in detail mode
		out.Append(outputs.Pango(
			pango.Icon("mdi-wifi").Alpha(0.6),
			pango.Text(i.SSID),
		))
		out.Append(outputs.Textf("%2.1fG", i.Frequency.Gigahertz()))
		out.Append(outputs.Pango(
			pango.Icon("mdi-access-point").Alpha(0.8),
			pango.Text(i.AccessPointMAC).Small(),
		))
		return out
	}), 1)

	vol := volume.New(alsa.DefaultMixer()).Output(func(v volume.Volume) bar.Output {
		if v.Mute {
			return outputs.
				Pango(pango.Icon("fa-volume-mute").Alpha(0.8), spacer, "MUT").
				Color(colors.Scheme("degraded"))
		}
		iconName := "off"
		pct := v.Pct()
		if pct > 66 {
			iconName = "up"
		} else if pct > 33 {
			iconName = "down"
		}
		return outputs.Pango(
			pango.Icon("fa-volume-"+iconName).Alpha(0.6),
			spacer,
			pango.Textf("%2d%%", pct),
		)
	})

	loadAvg := sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		out := outputs.Pango(
			pango.Icon("mdi-desktop-tower").Alpha(0.6),
			pango.Textf("%0.2f", s.Loads[0]),
		)
		// Load averages are unusually high for a few minutes after boot.
		if s.Uptime < 10*time.Minute {
			// so don't add colours until 10 minutes after system start.
			return out
		}
		threshold(out,
			s.Loads[0] > 128 || s.Loads[2] > 64,
			s.Loads[0] > 64 || s.Loads[2] > 32,
			s.Loads[0] > 32 || s.Loads[2] > 16,
		)
		out.OnClick(click.Left(func() {
			mainModalController.Toggle("sysinfo")
		}))
		return out
	})

	loadAvgDetail := sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		return pango.Textf("%0.2f %0.2f", s.Loads[1], s.Loads[2]).Smaller()
	})

	uptime := sysinfo.New().Output(func(s sysinfo.Info) bar.Output {
		u := s.Uptime
		var uptimeOut *pango.Node
		if u.Hours() < 24 {
			uptimeOut = pango.Textf("%d:%02d",
				int(u.Hours()), int(u.Minutes())%60)
		} else {
			uptimeOut = pango.Textf("%dd%02dh",
				int(u.Hours()/24), int(u.Hours())%24)
		}
		return pango.Icon("mdi-trending-up").Alpha(0.6).Concat(uptimeOut)
	})

	freeMem := meminfo.New().Output(func(m meminfo.Info) bar.Output {
		out := outputs.Pango(
			pango.Icon("material-memory").Alpha(0.8),
			format.IBytesize(m.Available()),
		)
		freeGigs := m.Available().Gigabytes()
		threshold(out,
			freeGigs < 0.5,
			freeGigs < 1,
			freeGigs < 2,
			freeGigs > 12)
		out.OnClick(click.Left(func() {
			mainModalController.Toggle("sysinfo")
		}))
		return out
	})

	swapMem := meminfo.New().Output(func(m meminfo.Info) bar.Output {
		return outputs.Pango(
			pango.Icon("mdi-swap-horizontal").Alpha(0.8),
			format.IBytesize(m["SwapTotal"]-m["SwapFree"]), spacer,
			pango.Textf("(% 2.0f%%)", (1-m.FreeFrac("Swap"))*100.0).Small(),
		)
	})

	temp := cputemp.New().
		RefreshInterval(2 * time.Second).
		Output(func(temp unit.Temperature) bar.Output {
			out := outputs.Pango(
				pango.Icon("mdi-fan").Alpha(0.6), spacer,
				pango.Textf("%2d℃", int(temp.Celsius())),
			)
			threshold(out,
				temp.Celsius() > 90,
				temp.Celsius() > 70,
				temp.Celsius() > 60,
			)
			return out
		})

	sub := netlink.Any()
	iface := sub.Get().Name
	sub.Unsubscribe()
	netsp := netspeed.New(iface).
		RefreshInterval(2 * time.Second).
		Output(func(s netspeed.Speeds) bar.Output {
			return outputs.Pango(
				pango.Icon("fa-upload").Alpha(0.5), spacer, pango.Textf("%7s", format.Byterate(s.Tx)),
				pango.Text(" ").Small(),
				pango.Icon("fa-download").Alpha(0.5), spacer, pango.Textf("%7s", format.Byterate(s.Rx)),
			)
		})

	net := netinfo.New().Output(func(i netinfo.State) bar.Output {
		if !i.Enabled() {
			return nil
		}
		if i.Connecting() || len(i.IPs) < 1 {
			return outputs.Text(i.Name).Color(colors.Scheme("degraded"))
		}
		return outputs.Group(outputs.Text(i.Name), outputs.Textf("%s", i.IPs[0]))
	})

	formatDiskSpace := func(i diskspace.Info, icon string) bar.Output {
		out := outputs.Pango(
			pango.Icon(icon).Alpha(0.7), spacer, format.IBytesize(i.Available))
		return threshold(out,
			i.Available.Gigabytes() < 1,
			i.AvailFrac() < 0.05,
			i.AvailFrac() < 0.1,
		)
	}

	rootDev := deviceForMountPath("/")
	var homeDiskspace bar.Module
	if deviceForMountPath(home()) != rootDev {
		homeDiskspace = diskspace.New(home()).Output(func(i diskspace.Info) bar.Output {
			return formatDiskSpace(i, "typecn-home-outline")
		})
	}
	rootDiskspace := diskspace.New("/").Output(func(i diskspace.Info) bar.Output {
		return formatDiskSpace(i, "fa-hdd")
	})

	mainDiskio := diskio.New(strings.TrimPrefix(rootDev, "/dev/")).
		Output(func(r diskio.IO) bar.Output {
			return pango.Icon("mdi-swap-vertical").
				Concat(spacer).
				ConcatText(format.IByterate(r.Total()))
		})

	mediaSummary, mediaDetail := split.New(media.Auto().Output(mediaFormatFunc), 1)

	ghNotify := github.New("%%GITHUB_CLIENT_ID%%", "%%GITHUB_CLIENT_SECRET%%").
		Output(func(n github.Notifications) bar.Output {
			if n.Total() == 0 {
				return nil
			}
			out := outputs.Group(
				pango.Icon("fab-github").
					Concat(spacer).
					ConcatTextf("%d", n.Total()))
			mentions := n["mention"] + n["team_mention"]
			if mentions > 0 {
				out.Append(spacer)
				out.Append(outputs.Pango(
					pango.Icon("mdi-bell").
						ConcatTextf("%d", mentions)).
					Urgent(true))
			}
			return out.Glue().OnClick(
				click.RunLeft("xdg-open", "https://github.com/notifications"))
		})

	mainModal := modal.New()
	sysMode := mainModal.Mode("sysinfo").
		SetOutput(makeIconOutput("mdi-chart-areaspline")).
		Add(loadAvg).
		Detail(loadAvgDetail, uptime).
		Add(freeMem).
		Detail(swapMem, temp)
	if homeDiskspace != nil {
		sysMode.Detail(homeDiskspace)
	}
	sysMode.Detail(rootDiskspace, mainDiskio)
	mainModal.Mode("network").
		SetOutput(makeIconOutput("mdi-ethernet")).
		Summary(wifiName).
		Detail(wifiDetails, net, netsp)
	mainModal.Mode("media").
		SetOutput(makeIconOutput("mdi-music-box")).
		Add(vol, mediaSummary).
		Detail(mediaDetail)
	mainModal.Mode("notifications").
		SetOutput(nil).
		Add(ghNotify)
	mainModal.Mode("battery").
		// Filled in by the battery module if one is available.
		SetOutput(nil).
		Summary(battSummary).
		Detail(battDetail)
	mainModal.Mode("weather").
		// Set to current conditions by the weather module.
		SetOutput(makeIconOutput("typecn-warning-outline")).
		Detail(wthr)
	mainModal.Mode("timezones").
		SetOutput(makeIconOutput("material-access-time")).
		Detail(makeTzClock("Seattle", "America/Los_Angeles")).
		Detail(makeTzClock("New York", "America/New_York")).
		Detail(makeTzClock("UTC", "Etc/UTC")).
		Detail(makeTzClock("Berlin", "Europe/Berlin")).
		Detail(makeTzClock("Tokyo", "Asia/Tokyo")).
		Add(localdate)

	var mm bar.Module
	mm, mainModalController = mainModal.Build()
	panic(barista.Run(mm, localtime))
}
