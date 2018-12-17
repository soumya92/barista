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

// Package media provides an i3bar module for an MPRIS-compatible media player.
package media // import "barista.run/modules/media"

import (
	"fmt"
	"strings"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/base/watchers/dbus"
	l "barista.run/logging"
	"barista.run/outputs"

	"golang.org/x/time/rate"
)

// PlaybackStatus represents the state of the media player.
type PlaybackStatus string

const (
	// Disconnected when the player is not running.
	Disconnected = PlaybackStatus("")
	// Playing when the player is actively playing a track.
	Playing = PlaybackStatus("Playing")
	// Paused when some media is selected but paused.
	Paused = PlaybackStatus("Paused")
	// Stopped when no media is selected or playing.
	Stopped = PlaybackStatus("Stopped")
)

// Info represents the current information from the media player.
type Info struct {
	PlayerName     string
	PlaybackStatus PlaybackStatus
	Shuffle        bool
	// From Metadata
	Length      time.Duration
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	// Although ArtURL cannot be used in the module output, it can still be
	// used for notifications or colour extraction.
	ArtURL string
	// Position is computed from the last known position and rate,
	// since position updates don't trigger any updates.
	lastUpdated  time.Time
	lastPosition time.Duration
	rate         float64
	// TrackID is used to determine whether the metadata change was caused by
	// a track change or a metadata update to the current track.
	// unexported because it won't be set when position is not being tracked.
	trackID string
	// A method to forward DBus calls, allowing control of the player.
	call func(string, ...interface{}) ([]interface{}, error)
}

// Module represents a bar.Module that displays media information
// from an MPRIS-compatible media player.
type Module struct {
	playerName value.Value // of string
	outputFunc value.Value // of func(Info) bar.Output
}

// New constructs an instance of the media module for the given player.
func New(player string) *Module {
	m := new(Module)
	m.playerName.Set(player)
	l.Label(m, player)
	l.Register(m, "playerName", "outputFunc")
	// Default output is just the currently playing track.
	m.Output(func(i Info) bar.Output {
		if i.Playing() {
			return outputs.Repeat(func(time.Time) bar.Output {
				return outputs.Textf("%v: %s", i.TruncatedPosition("s"), i.Title)
			}).Every(time.Second)
		}
		if i.Connected() {
			return outputs.Text(i.Title)
		}
		return nil
	})
	return m
}

// Player sets the name of the player to track. This will disconnect the module
// from the previous player.
func (m *Module) Player(player string) *Module {
	l.Label(m, player)
	m.playerName.Set(player)
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RepeatingOutput configures a module to display the output of a user-defined
// function, automatically repeating it every second while playing.
func (m *Module) RepeatingOutput(outputFunc func(Info) bar.Output) *Module {
	return m.Output(func(i Info) bar.Output {
		if i.Playing() {
			return outputs.Repeat(func(time.Time) bar.Output {
				return outputFunc(i)
			}).Every(time.Second)
		}
		return outputFunc(i)
	})
}

// AutoModule is a media module that automatically switches to the newest media
// player seen on D-Bus.
type AutoModule struct {
	module   *Module
	excluded map[string]bool
}

// Auto constructs an instance of the media module that shows the most recently
// connected player (based on D-Bus name acquisition). It can optionally ignore
// one or more named players from this detection.
func Auto(excluding ...string) *AutoModule {
	excluded := map[string]bool{}
	for _, e := range excluding {
		excluded[e] = true
	}
	m := &AutoModule{New(""), excluded}
	l.Attach(m.module, m, "~auto")
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *AutoModule) Output(outputFunc func(Info) bar.Output) *AutoModule {
	m.module.Output(outputFunc)
	return m
}

// RepeatingOutput configures a module to display the output of a user-defined
// function, automatically repeating it every second while playing.
func (m *AutoModule) RepeatingOutput(outputFunc func(Info) bar.Output) *AutoModule {
	m.module.RepeatingOutput(outputFunc)
	return m
}

// Throttle seek calls to once every ~50ms to allow more control
// and work around some programs that cannot handle rapid updates.
var seekLimiter = rate.NewLimiter(rate.Every(50*time.Millisecond), 1)

// defaultClickHandler provides useful behaviour out of the box,
// Click to play/pause, scroll to seek, and back/forward to switch tracks.
func defaultClickHandler(i Info) func(bar.Event) {
	return func(e bar.Event) {
		switch e.Button {
		case bar.ButtonLeft:
			i.PlayPause()
		case bar.ScrollDown, bar.ScrollRight:
			if seekLimiter.Allow() {
				i.Seek(time.Second)
			}
		case bar.ButtonBack:
			i.Previous()
		case bar.ScrollUp, bar.ScrollLeft:
			if seekLimiter.Allow() {
				i.Seek(-time.Second)
			}
		case bar.ButtonForward:
			i.Next()
		}
	}
}

// Replaced in tests.
var busType = dbus.Session

// Stream sets up d-bus connections and starts the module.
func (m *Module) Stream(s bar.Sink) {
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()

	playerName := m.playerName.Get().(string)
	nextPlayerName, done := m.playerName.Subscribe()
	defer done()

	w, info := subscribeToPlayer(playerName)
	for {
		s.Output(outputs.Group(outputFunc(info)).
			OnClick(defaultClickHandler(info)))
		select {
		case <-nextPlayerName:
			w.Unsubscribe()
			playerName = m.playerName.Get().(string)
			w, info = subscribeToPlayer(playerName)
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
		case u := <-w.Updates:
			for k, v := range u {
				info.set(k, v[1])
			}
		}
	}
}

// subscribes to the player with the given mpris name via dbus. Returns a dbus
// properties watcher and the intial media info from the player.
func subscribeToPlayer(playerName string) (*dbus.PropertiesWatcher, Info) {
	w := dbus.WatchProperties(busType,
		fmt.Sprintf("org.mpris.MediaPlayer2.%s", playerName),
		"/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player").
		Add("Rate", "Shuffle", "PlaybackStatus", "Metadata").
		FetchOnSignal("Position").
		AddSignalHandler("Seeked", func(s *dbus.Signal, _ dbus.Fetcher) map[string]interface{} {
			return map[string]interface{}{"Position": s.Body[0]}
		})
	info := Info{PlayerName: playerName, call: w.Call}
	for k, v := range w.Get() {
		info.set(k, v)
	}
	l.Fine("subscribe to %s: %v", playerName, info)
	return w, info
}

// Stream starts the module and the D-Bus listener for media player name
// acquisitions and releases.
func (m *AutoModule) Stream(s bar.Sink) {
	w := dbus.WatchNameOwners(busType, "org.mpris.MediaPlayer2")
	defer w.Unsubscribe()
	ownerStack := []string{}
	for k := range w.GetOwners() {
		if len(ownerStack) == 0 {
			l.Fine("%s, starting with %s", l.ID(m), k)
			m.playerName(k)
		}
		ownerStack = append(ownerStack, k)
	}
	go m.listenForPlayerUpdates(w.Updates, ownerStack)
	m.module.Stream(s)
}

func (m *AutoModule) playerName(dbusName string) {
	m.module.Player(strings.TrimPrefix(dbusName, "org.mpris.MediaPlayer2."))
}

func (m *AutoModule) listenForPlayerUpdates(updates <-chan dbus.NameOwnerChange, ownerStack []string) {
	for u := range updates {
		if m.excluded[strings.TrimPrefix(u.Name, "org.mpris.MediaPlayer2.")] {
			continue
		}
		if u.Owner != "" {
			// New player, switch to it.
			l.Fine("%s: switching to new player %s", l.ID(m), u.Name)
			m.playerName(u.Name)
			ownerStack = append(ownerStack, u.Name)
			continue
		}
		for i, n := range ownerStack {
			if n == u.Name {
				ownerStack = append(ownerStack[:i], ownerStack[i+1:]...)
			}
		}
		l.Fine("player %s disconnected", u.Name)
		curr := m.module.playerName.Get().(string)
		if u.Name == "org.mpris.MediaPlayer2."+curr {
			l.Fine("%s: current player %s disconnected", l.ID(m), u.Name)
			count := len(ownerStack)
			if count > 0 {
				l.Fine("%s: switching to %s", l.ID(m), ownerStack[count-1])
				m.playerName(ownerStack[count-1])
			}
		}
	}
}
