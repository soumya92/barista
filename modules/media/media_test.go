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

package media

import (
	"testing"
	"time"

	"github.com/soumya92/barista/bar"
	dbusWatcher "github.com/soumya92/barista/base/watchers/dbus"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
	"golang.org/x/time/rate"

	"github.com/godbus/dbus/v5"
	"github.com/stretchr/testify/require"
)

func init() {
	busType = dbusWatcher.Test
}

type methodCall struct {
	name string
	arg  interface{}
}

func TestMedia(t *testing.T) {
	// To allow -count >1 to work.
	seekLimiter = rate.NewLimiter(rate.Every(50*time.Millisecond), 1)

	testBar.New(t)
	bus := dbusWatcher.SetupTestBus()
	srv := bus.RegisterService("org.mpris.MediaPlayer2.testplayer")
	obj := srv.Object("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	obj.SetProperties(map[string]interface{}{
		"Position":       180 * 1000 * 1000,
		"PlaybackStatus": "Playing",
		"Rate":           1.0,
		"Metadata": map[string]dbus.Variant{
			"xesam:title":  dbus.MakeVariant("Title"),
			"xesam:artist": dbus.MakeVariant([]string{"Artist1", "Artist2"}),
		},
	}, dbusWatcher.SignalTypeNone)
	calls := make(chan methodCall, 10)

	pl := New("testplayer")
	testBar.Run(pl)
	testBar.NextOutput("on start").AssertText([]string{"3m0s: Title"})

	obj.SetPropertyForTest("PlaybackStatus", "Paused", dbusWatcher.SignalTypeChanged)
	testBar.NextOutput("on props change").AssertText([]string{"Title"})

	obj.SetPropertyForTest("Metadata", map[string]dbus.Variant{
		"xesam:title": dbus.MakeVariant("foo"),
	}, dbusWatcher.SignalTypeChanged)
	testBar.NextOutput("on title change").AssertText([]string{"foo"})

	srv1 := bus.RegisterService()
	obj = srv1.Object("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	obj.SetPropertyForTest("PlaybackStatus", "Paused", dbusWatcher.SignalTypeNone)
	obj.OnElse(func(method string, args ...interface{}) ([]interface{}, error) {
		c := methodCall{name: method}
		if len(args) > 0 {
			c.arg = args[0]
		}
		calls <- c
		return nil, nil
	})
	testBar.AssertNoOutput("On unrelated service changes")

	srv.Unregister()
	testBar.NextOutput("on service shutdown").AssertEmpty()

	srv1.AddName("org.mpris.MediaPlayer2.testplayer")
	testBar.NextOutput("on service move").AssertText([]string{""},
		"Does not show stale title")

	obj.SetProperties(map[string]interface{}{
		"PlaybackStatus": "Paused",
		"Shuffle":        true,
		"Rate":           1.0,
		"Metadata": map[string]dbus.Variant{
			"xesam:title":   dbus.MakeVariant("Song"),
			"mpris:trackid": dbus.MakeVariant("a"),
			"mpris:ArtURL":  dbus.MakeVariant("file:///tmp/art.webp"),
		},
	}, dbusWatcher.SignalTypeInvalidated)
	out := testBar.NextOutput("on properties change")
	out.AssertText([]string{"Song"}, "Still paused")

	out.At(0).Click(bar.Event{Button: bar.ButtonLeft})
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.PlayPause", nil},
		<-calls, "On left click")

	obj.SetPropertyForTest("PlaybackStatus", "Playing", dbusWatcher.SignalTypeChanged)
	out = testBar.NextOutput("on playstate change")
	out.AssertText([]string{"0s: Song"})

	out.At(0).Click(bar.Event{Button: bar.ScrollLeft})
	out.At(0).Click(bar.Event{Button: bar.ScrollRight})
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Seek", int64(-1000 * 1000)},
		<-calls, "On scroll left")
	select {
	case <-calls:
		require.Fail(t, "Unexpected method call",
			"Rate limiter should not allow second call")
	case <-time.After(50 * time.Millisecond):
	}

	obj.Emit("Seeked", 99*1000*1000)
	out = testBar.NextOutput("on seek")
	out.AssertText([]string{"1m39s: Song"})

	out.At(0).Click(bar.Event{Button: bar.ButtonBack})
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Previous", nil},
		<-calls, "On back click")

	obj.SetPropertyForTest("Metadata", map[string]dbus.Variant{
		"xesam:title":   dbus.MakeVariant("Title"),
		"xesam:artist":  dbus.MakeVariant([]string{"Artist1", "Artist2"}),
		"xesam:album":   dbus.MakeVariant("Album"),
		"mpris:trackid": dbus.MakeVariant("2"),
	}, dbusWatcher.SignalTypeInvalidated)
	testBar.NextOutput("on metadata update").AssertText([]string{"0s: Title"})

	var lastInfo Info
	pl.RepeatingOutput(func(i Info) bar.Output {
		lastInfo = i
		return outputs.Textf("[%s, %v] %s - %s",
			i.PlaybackStatus, i.TruncatedPosition("k"), i.Title, i.Artist)
	})
	out = testBar.NextOutput("on output format change")
	out.AssertText([]string{"[Playing, 0s] Title - Artist1"})

	out.At(0).Click(bar.Event{Button: bar.ButtonForward})
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Next", nil},
		<-calls, "On click with custom output")

	obj.SetPropertyForTest("Metadata", map[string]dbus.Variant{
		"xesam:title":       dbus.MakeVariant("Song"),
		"mpris:trackid":     dbus.MakeVariant("3"),
		"xesam:albumArtist": dbus.MakeVariant([]string{"Person1", "Person2"}),
		"mpris:length":      dbus.MakeVariant(180 * 1000 * 1000),
	}, dbusWatcher.SignalTypeInvalidated)
	out = testBar.NextOutput("on metadata update")
	out.AssertText([]string{"[Playing, 0s] Song - "})

	timing.AdvanceBy(time.Second)
	out = testBar.NextOutput("on time passing")
	out.AssertText([]string{"[Playing, 1s] Song - "})

	obj.SetPropertyForTest("PlaybackStatus", "Paused", dbusWatcher.SignalTypeChanged)
	out = testBar.NextOutput("on pause")
	out.AssertText([]string{"[Paused, 1s] Song - "})

	timing.AdvanceBy(time.Second)
	testBar.AssertNoOutput("on time passing, but paused")

	out.At(0).Click(bar.Event{Button: bar.ScrollDown})
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Seek", int64(1000 * 1000)},
		<-calls, "On scroll with custom output")

	obj.Emit("Seeked", 2*1000*1000)
	testBar.NextOutput("on seek").AssertText([]string{"[Paused, 2s] Song - "})

	obj.SetPropertyForTest("PlaybackStatus", "Stopped", dbusWatcher.SignalTypeChanged)
	testBar.NextOutput("on stop").AssertText([]string{"[Stopped, 0s] Song - "})

	obj.SetPropertyForTest("PlaybackStatus", "Playing", dbusWatcher.SignalTypeChanged)
	testBar.NextOutput("on play").AssertText([]string{"[Playing, 0s] Song - "})

	lastInfo.Stop()
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Stop", nil},
		<-calls, "Info.Stop()")
	lastInfo.Play()
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Play", nil},
		<-calls, "Info.Stop()")
	lastInfo.Pause()
	require.Equal(t,
		methodCall{"org.mpris.MediaPlayer2.Player.Pause", nil},
		<-calls, "Info.Stop()")

	require.True(t, lastInfo.Connected(), "Playing is Connected()")
	require.True(t, lastInfo.Playing(), "Playing == Playing()")
	require.False(t, lastInfo.Paused(), "Playing != Paused()")
	require.False(t, lastInfo.Stopped(), "Playing != Stopped()")
}

func TestAutoMedia(t *testing.T) {
	testBar.New(t)
	bus := dbusWatcher.SetupTestBus()

	srvA := bus.RegisterService("org.mpris.MediaPlayer2.A")
	objA := srvA.Object("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	objA.SetProperties(map[string]interface{}{
		"PlaybackStatus": "Paused",
		"Metadata": map[string]dbus.Variant{
			"xesam:title":  dbus.MakeVariant("TitleA"),
			"xesam:artist": dbus.MakeVariant([]string{"Artist1", "Artist2"}),
		},
	}, dbusWatcher.SignalTypeNone)

	auto := Auto("Ignored")
	testBar.Run(auto)
	testBar.NextOutput("on start").AssertText([]string{"TitleA"})

	srvB := bus.RegisterService()
	objB := srvB.Object("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	objB.SetProperties(map[string]interface{}{
		"PlaybackStatus": "Stopped",
		"Metadata": map[string]dbus.Variant{
			"xesam:title": dbus.MakeVariant("TitleB"),
		},
	}, dbusWatcher.SignalTypeNone)
	srvB.AddName("org.mpris.MediaPlayer2.B")

	testBar.NextOutput("on new player").AssertText([]string{"TitleB"})

	objA.SetProperties(map[string]interface{}{"PlaybackStatus": "Stopped"},
		dbusWatcher.SignalTypeChanged)
	testBar.AssertNoOutput("on change of previous player")

	objB.SetProperties(map[string]interface{}{"PlaybackStatus": "Paused"},
		dbusWatcher.SignalTypeChanged)
	testBar.NextOutput("on active player state change").
		AssertText([]string{"TitleB"})

	auto.RepeatingOutput(func(i Info) bar.Output {
		return outputs.Textf("[%v] %s - %s", i.PlaybackStatus, i.Title, i.Artist)
	})
	testBar.NextOutput("on output format change").
		AssertText([]string{"[Paused] TitleB - "})

	srvC := bus.RegisterService()
	objC := srvC.Object("/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player")
	objC.SetProperties(map[string]interface{}{
		"PlaybackStatus": "Stopped",
		"Metadata": map[string]dbus.Variant{
			"xesam:title":  dbus.MakeVariant("TitleC"),
			"xesam:artist": dbus.MakeVariant([]string{"Artist"}),
		},
	}, dbusWatcher.SignalTypeNone)
	srvC.AddName("org.mpris.MediaPlayer2.C")
	testBar.NextOutput("on new player").
		AssertText([]string{"[Stopped] TitleC - Artist"})

	auto.Output(func(i Info) bar.Output {
		return outputs.Textf("%v: %s", i.PlaybackStatus, i.Title)
	})
	testBar.NextOutput("on output format change").
		AssertText([]string{"Stopped: TitleC"})

	srvIgn := bus.RegisterService("org.mpris.MediaPlayer2.Ignored")
	testBar.AssertNoOutput("On ignored player connection")

	objB.SetProperties(map[string]interface{}{"PlaybackStatus": "Playing"},
		dbusWatcher.SignalTypeChanged)
	testBar.AssertNoOutput("on inactive player change")
	srvB.Unregister()
	testBar.AssertNoOutput("on inactive player disconnect")

	srvC.Unregister()
	testBar.
		Drain(time.Second, "on active player disconnect").
		AssertText([]string{"Stopped: TitleA"})

	srvIgn.Unregister()
	testBar.AssertNoOutput("On ignored player disconnection")

	srvA.Unregister()
	testBar.NextOutput("on disconnect").
		AssertText([]string{": "})
}

func TestDbusLongAndFloats(t *testing.T) {
	for _, tc := range []struct {
		val      interface{}
		asDouble float64
		asLong   int64
	}{
		{1.1, 1.1, 1},
		{2, 2.0, 2},
		{float32(3.3), 3.3, 3},
		{dbus.MakeVariant(4.4), 4.4, 4},
		{uint(5), 5.0, 5},
		{int64(6), 6.0, 6},
		{int32(7), 7.0, 7},
		{uint8(8), 8.0, 8},
		{dbus.MakeVariant(uint(9)), 9.0, 9},
		{dbus.MakeVariant(int64(10)), 10.0, 10},
		{dbus.MakeVariant(int32(11)), 11.0, 11},
		{dbus.MakeVariant(uint8(12)), 12.0, 12},
		{"foo", 0.0, 0},
		{dbus.MakeVariant("baz"), 0.0, 0},
		{"13.34", 0.0, 0},
		{dbus.MakeVariant("14.45"), 0.0, 0},
		{dbus.MakeVariant([]float64{15.8}), 0.0, 0},
	} {
		require.InDelta(t, tc.asDouble, getDouble(tc.val), 0.001,
			"getDouble(%v) == %v", tc.val, tc.asDouble)
		require.Equal(t, tc.asLong, getLong(tc.val),
			"getLong(%v) == %v", tc.val, tc.asLong)
	}
}
