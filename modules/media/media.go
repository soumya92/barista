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

	"github.com/godbus/dbus"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
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

// Module is the public interface for a media module.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to control the media player.
type Module interface {
	base.Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Info) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OnClick sets a click handler for the module.
	OnClick(func(Info, Controller, bar.Event)) Module
}

type module struct {
	*base.Base
	playerName string
	outputFunc func(Info) bar.Output
	// player state, updated from dbus signals.
	info Info
	// To simplify adding/removing matches and querying metadata,
	// store references to bus and player dbus objects.
	player *mprisPlayer
	// An additional update every second while music is playing
	// to keep the position up to date.
	positionScheduler base.Scheduler
}

// New constructs an instance of the media module for the given player.
func New(player string) Module {
	m := &module{
		Base:       base.New(),
		playerName: player,
	}
	// Set default click handler in New(), can be overridden later.
	m.OnClick(DefaultClickHandler)
	// Default output template that's just the currently playing track.
	m.OutputTemplate(outputs.TextTemplate(`{{if .Connected}}{{.Title}}{{end}}`))
	return m
}

func (m *module) OutputFunc(outputFunc func(Info) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

func (m *module) OnClick(f func(Info, Controller, bar.Event)) Module {
	if f == nil {
		m.Base.OnClick(nil)
		return m
	}
	m.Base.OnClick(func(e bar.Event) {
		f(m.info, m.player, e)
	})
	return m
}

// DefaultClickHandler provides useful behaviour out of the box,
// Click to play/pause, scroll to seek, and back/forward to switch tracks.
func DefaultClickHandler(i Info, c Controller, e bar.Event) {
	switch e.Button {
	case bar.ButtonLeft:
		c.PlayPause()
	case bar.ScrollDown, bar.ScrollRight:
		c.Seek(time.Second)
	case bar.ButtonBack:
		c.Previous()
	case bar.ScrollUp, bar.ScrollLeft:
		c.Seek(-time.Second)
	case bar.ButtonForward:
		c.Next()
	}
}

// Stream sets up d-bus connections and then returns the output
// channel from the base module. This allows us to skip error
// checking in the update function since we're guaranteed that
// the update function will only be called if there were no errors
// during startup.
func (m *module) Stream() (ch <-chan bar.Output) {
	ch = m.Base.Stream()
	// Need a private bus in-case other modules (or other instances of media) are
	// using dbus as well. Since we rely on (Add|Remove)Match and Signal,
	// we cannot share the session bus.
	sessionBus, err := dbus.SessionBusPrivate()
	if m.Error(err) {
		return
	}
	// Need to handle auth and handshake ourselves for private sessions buses.
	if err := sessionBus.Auth(nil); m.Error(err) {
		return
	}
	if err := sessionBus.Hello(); m.Error(err) {
		return
	}
	m.player = newMprisPlayer(sessionBus, m.playerName, &m.info)
	if m.Error(m.player.err) {
		return
	}
	// If we made it this far, set the update function.
	m.OnUpdate(m.update)
	// Initial output.
	m.Update()

	// Since the channel is shared with method call responses,
	// we need a buffer to prevent deadlocks.
	// The buffer value of 10 is based on the dbus signal example,
	// and is also used in corp/access/credkit.
	c := make(chan *dbus.Signal, 10)
	sessionBus.Signal(c)
	go m.listen(c)
	return
}

// listen handles dbus signals from the player, and updates
// the module output when necessary.
func (m *module) listen(c <-chan *dbus.Signal) {
	for v := range c {
		updated, err := m.player.handleDbusSignal(v)
		if m.Error(err) {
			continue
		}
		if updated {
			m.Update()
		}
	}
}

func (m *module) update() {
	m.Output(m.outputFunc(m.info))
	m.positionScheduler.Stop()
	if m.info.PlaybackStatus == Playing {
		m.positionScheduler = m.UpdateEvery(time.Second)
	}
}
