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
package media

import (
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/godbus/dbus"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/timing"
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
	Controller
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
}

// Paused returns true if the media player is connected and paused.
func (i Info) Paused() bool {
	return i.PlaybackStatus == Paused
}

// Playing returns true if the media player is connected and playing media.
func (i Info) Playing() bool {
	return i.PlaybackStatus == Playing
}

// Stopped returns true if the media player is connected but stopped.
func (i Info) Stopped() bool {
	return i.PlaybackStatus == Stopped
}

// Connected returns true if the media player is connected.
func (i Info) Connected() bool {
	return i.PlaybackStatus != Disconnected
}

// Position computes the current track position
// based on the last update from the media player.
func (i Info) Position() time.Duration {
	if i.PlaybackStatus == Paused {
		// If paused, then the position is not advancing.
		return i.lastPosition
	}
	elapsed := time.Since(i.lastUpdated)
	return i.lastPosition + time.Duration(float64(elapsed)*i.rate)
}

// TruncatedPosition truncates the current position to the given unit,
// to avoid unnecessary decimals (e.g. 1m3.0032s becomes 1m3s for "s")
func (i Info) TruncatedPosition(unit string) string {
	dur, err := time.ParseDuration("1" + unit)
	if err != nil {
		dur = time.Second
	}
	rounded := (i.Position() + dur/2) / dur * dur
	s := rounded.String()
	if i := strings.LastIndex(s, unit); i != -1 {
		return s[0 : i+len(unit)]
	}
	return s
}

// snapshotPosition snapshots the playback position,
// useful when updates to rate or playback status would yield incorrect results.
func (i *Info) snapshotPosition() {
	now := time.Now()
	elapsed := now.Sub(i.lastUpdated)
	i.lastPosition += time.Duration(float64(elapsed) * i.rate)
	i.lastUpdated = now
}

// Controller provides an interface to control the media player,
// used in the click handler.
type Controller interface {
	// Play resumes playback of the current track.
	Play()

	// Pause pauses the track. No effect if not playing.
	Pause()

	// PlayPause toggles between play and pause on the media player.
	PlayPause()

	// Stop stops and clears the currently playing track.
	Stop()

	// Next switches to the next track
	Next()

	// Previous switches to the previous track (or restarts the current track).
	// Implementation is player-dependent.
	Previous()

	// Seek seeks to the specified offset from the current position.
	// Use negative durations to seek backwards.
	Seek(offset time.Duration)
}

// Module represents a bar.Module that displays media information
// from an MPRIS-compatible media player.
type Module struct {
	playerName string
	outputFunc base.Value // of func(Info) bar.Output

	// player state, updated from dbus signals.
	info base.Value // of Info

	// To simplify adding/removing matches and querying metadata,
	// store references to bus and player dbus objects.
	player *mprisPlayer
}

// New constructs an instance of the media module for the given player.
func New(player string) *Module {
	m := &Module{playerName: player}
	l.Label(m, player)
	l.Register(m, "outputFunc", "clickHandler", "info")
	// Default output template that's just the currently playing track.
	m.Template(`{{if .Connected}}{{.Title}}{{end}}`)
	return m
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Template configures a module to display the output of a template.
func (m *Module) Template(template string) *Module {
	base.Template(template, m.Output)
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

// Stream sets up d-bus connections and starts the module.
func (m *Module) Stream(s bar.Sink) {
	// Need a private bus in-case other modules (or other instances of media) are
	// using dbus as well. Since we rely on (Add|Remove)Match and Signal,
	// we cannot share the session bus.
	sessionBus, err := dbus.SessionBusPrivate()
	if s.Error(err) {
		return
	}
	// Need to handle auth and handshake ourselves for private sessions buses.
	if err := sessionBus.Auth(nil); s.Error(err) {
		return
	}
	if err := sessionBus.Hello(); s.Error(err) {
		return
	}

	info := Info{}
	outputFunc := m.outputFunc.Get().(func(Info) bar.Output)

	m.player = newMprisPlayer(sessionBus, m.playerName, &info)
	if s.Error(m.player.err) {
		return
	}
	m.info.Set(info)

	positionUpdater := timing.NewScheduler()
	l.Attach(m, positionUpdater, "positionUpdater")
	// If currently playing, also start the position updater.
	if info.Playing() {
		positionUpdater.Every(time.Second)
	}

	// Since the channel is shared with method call responses,
	// we need a buffer to prevent deadlocks.
	// The buffer value of 10 is based on the dbus signal example,
	// and is also used in corp/access/credkit.
	dbusCh := make(chan *dbus.Signal, 10)
	sessionBus.Signal(dbusCh)

	for {
		select {
		case <-m.outputFunc.Next():
			outputFunc = m.outputFunc.Get().(func(Info) bar.Output)
			info.Controller = m.player
			s.Output(outputs.Group(outputFunc(info)).
				OnClick(defaultClickHandler(info)))
		case v := <-dbusCh:
			updates, err := m.player.handleDbusSignal(v)
			if s.Error(err) {
				return
			}
			l.Log("%s: updated %#v from dbus signal", l.ID(m), updates)
			if updates.playingState {
				if info.Playing() {
					positionUpdater.Every(time.Second)
				} else {
					positionUpdater.Stop()
				}
			}
			if updates.any() {
				m.info.Set(info)
				info.Controller = m.player
				s.Output(outputs.Group(outputFunc(info)).
					OnClick(defaultClickHandler(info)))
			}
		case <-positionUpdater.Tick():
			info.Controller = m.player
			s.Output(outputs.Group(outputFunc(info)).
				OnClick(defaultClickHandler(info)))
		}
	}
}
