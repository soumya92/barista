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
	playerName string
	outputFunc value.Value // of func(Info) bar.Output
}

// New constructs an instance of the media module for the given player.
func New(player string) *Module {
	m := &Module{playerName: player}
	l.Label(m, player)
	l.Register(m, "outputFunc")
	// Default output is just the currently playing track.
	m.Output(func(i Info) bar.Output {
		if i.Playing() {
			return outputs.Repeat(func(t time.Time) bar.Output {
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

	w := dbus.WatchProperties(busType,
		fmt.Sprintf("org.mpris.MediaPlayer2.%s", m.playerName),
		"/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player").
		Add("Rate", "Shuffle", "PlaybackStatus", "Metadata").
		FetchOnSignal("Position").
		AddSignalHandler("Seeked", func(s *dbus.Signal, _ dbus.Fetcher) map[string]interface{} {
			return map[string]interface{}{"Position": s.Body[0]}
		})

	info := Info{call: w.Call}
	for k, v := range w.Get() {
		info.set(k, v)
	}

	for {
		s.Output(outputs.Group(outputFunc(info)).
			OnClick(defaultClickHandler(info)))
		select {
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
		case u := <-w.Updates:
			for k, v := range u {
				info.set(k, v[1])
			}
		}
	}
}
