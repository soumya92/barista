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

package media

import (
	"fmt"
	"time"

	"github.com/godbus/dbus"
)

// mprisPlayer stores the dbus, player object, and an error together,
// to simplify checking for errors after each call.
type mprisPlayer struct {
	bus    dbus.BusObject
	player dbus.BusObject
	info   *Info
	err    error
}

func newMprisPlayer(sessionBus *dbus.Conn, playerName string, info *Info) *mprisPlayer {
	// dbus name for the player.
	dest := fmt.Sprintf("org.mpris.MediaPlayer2.%s", playerName)
	// Get the dbus objects for the session bus and the player.
	player := &mprisPlayer{
		player: sessionBus.Object(dest, "/org/mpris/MediaPlayer2"),
		bus:    sessionBus.BusObject(),
		info:   info,
	}
	// Check if the player is already running.
	res, ok := player.Call(methodNameHasOwner, dest)
	if ok && res.(bool) {
		// Player is running.
		res, ok := player.Call(methodGetNameOwner, dest)
		if ok {
			// Get initial media info.
			player.getInitialInfo()
			// Add signal matches for metadata change and seek.
			player.addMatches(res.(string))
		}
	}
	// If the player is not running, do nothing,
	// and the NameOwnerChanged listener will take care of it.
	// Add listeners for player startup/shutdown to keep track of it's bus id.
	player.Call(methodAddMatch, signalNameOwnerChanged.buildMatchString("", dest))
	return player
}

func (m *mprisPlayer) addMatches(sender string) {
	m.Call(methodAddMatch, signalSeeked.buildMatchString(sender))
	m.Call(methodAddMatch, signalPropChanged.buildMatchString(sender, mprisInterface))
}

func (m *mprisPlayer) removeMatches(sender string) {
	m.Call(methodRemoveMatch, signalSeeked.buildMatchString(sender))
	m.Call(methodRemoveMatch, signalPropChanged.buildMatchString(sender, mprisInterface))
}

func (m *mprisPlayer) Play() {
	m.Call(mprisPlay)
}

func (m *mprisPlayer) Pause() {
	m.Call(mprisPause)
}

func (m *mprisPlayer) PlayPause() {
	m.Call(mprisPlayPause)
}

func (m *mprisPlayer) Stop() {
	m.Call(mprisStop)
}

func (m *mprisPlayer) Next() {
	m.Call(mprisNext)
}

func (m *mprisPlayer) Previous() {
	m.Call(mprisPrev)
}

func (m *mprisPlayer) Seek(offset time.Duration) {
	micros := int64(offset / time.Microsecond)
	m.Call(mprisSeek, micros)
}

// Call forwards a method call to either the bus or the player as appropriate,
// and returns the first returned value (or nil if nothing was returned).
func (m *mprisPlayer) Call(method name, args ...interface{}) (interface{}, bool) {
	if m.err != nil {
		return nil, false
	}
	var call *dbus.Call
	if method.iface == mprisInterface {
		// m.player's interface != mprisInterface, so full method name is required.
		call = m.player.Call(method.String(), 0, args...)
	} else {
		call = m.bus.Call(method.member, 0, args...)
	}
	m.err = call.Err
	if m.err != nil {
		return nil, false
	}
	if len(call.Body) > 0 {
		return call.Body[0], true
	}
	return nil, true
}

// getInitialInfo gets the initial media info from a player.
func (m *mprisPlayer) getInitialInfo() {
	i := m.infoReader(m)
	i.updatePlaybackStatus()
	i.updateMetadata()
	i.updateShuffle()
	i.updatePosition()
	i.updateRate()
}

// updates keeps track of the types of updates caused by a dbus signal.
type updates struct {
	// Includes position due to explicit seek and play state.
	position bool
	// Includes all metadata, including art.
	metadata bool
	// Only when going from playing <-> something else. That is,
	// stopped -> paused will not mark this true.
	playingState bool
}

// any returns true if any updates occurred when handing the dbus signal.
func (u updates) any() bool {
	return u.position || u.metadata || u.playingState
}

// handleDbusSignal handles dbus signals for track changes, seeking, and
// name owner changes (player appearing on or disappearing from session bus).
func (m *mprisPlayer) handleDbusSignal(signal *dbus.Signal) (updates, error) {
	switch signal.Name {
	case signalPropChanged.String():
		i := m.infoReader(dbusMap(signal.Body[1].(map[string]dbus.Variant)))
		i.updatePlaybackStatus()
		i.updateMetadata()
		i.updateShuffle()
		i.updatePosition()
		i.updateRate()
		return i.updates, m.err

	case signalSeeked.String():
		i := m.infoReader(dbusMap{
			mprisPosition.member: dbus.MakeVariant(signal.Body[0]),
		})
		i.updatePosition()
		return i.updates, m.err

	case signalNameOwnerChanged.String():
		oldName := signal.Body[1].(string)
		newName := signal.Body[2].(string)
		if len(oldName) > 0 {
			m.removeMatches(oldName)
		} else {
			// Clear cached info on new name acquisition, since some players
			// don't send empty info on start.
			*m.info = Info{}
		}
		if len(newName) > 0 {
			m.addMatches(newName)
			m.getInitialInfo()
		} else {
			// Empty newName => player disconnected from dbus.
			m.info.PlaybackStatus = Disconnected
			return updates{true, true, true}, m.err
		}
	}
	return updates{}, m.err
}

func (m *mprisPlayer) infoReader(getter dbusGetter) infoReader {
	return infoReader{
		Info:       m.info,
		dbusGetter: getter,
	}
}

// dbusGetter allows getting named dbus properties.
type dbusGetter interface {
	Get(name) (interface{}, bool)
}

// dbusMap implements dbusGetter for a map of string -> variants.
type dbusMap map[string]dbus.Variant

func (i dbusMap) Get(n name) (interface{}, bool) {
	if v, ok := i[n.member]; ok {
		return v.Value(), true
	}
	return nil, false
}

// Get gets a player property from the mpris player object.
// This satisfies the dbusGetter interface via queries to the player.
func (m *mprisPlayer) Get(prop name) (interface{}, bool) {
	if m.err != nil {
		return nil, false
	}
	var v dbus.Variant
	v, m.err = m.player.GetProperty(prop.String())
	if m.err == nil {
		return v.Value(), true
	}
	return nil, false
}

// infoReader providers helper methods to update media info
// from dbus signals or properties.
type infoReader struct {
	*Info
	dbusGetter
	updates updates
}

func (i *infoReader) updatePlaybackStatus() {
	status, ok := i.Get(mprisStatus)
	if !ok {
		return
	}
	oldState := i.PlaybackStatus
	i.PlaybackStatus = PlaybackStatus(status.(string))
	if i.PlaybackStatus == oldState {
		return
	}
	i.updates.position = true
	// Mark playing state as changed if we went from playing to something else,
	// or if we went from anything else to playing.
	// We use this to determine whether the scheduler for the current position
	// needs to be updated.
	if i.PlaybackStatus == Playing || oldState == Playing {
		i.updates.playingState = true
	}
	switch i.PlaybackStatus {
	case Playing:
		if oldState == Paused {
			// If we resumed playing, keep current position
			// but mark it as just updated.
			i.lastUpdated = time.Now()
		}
	case Paused:
		if oldState == Playing {
			// If we paused from playing, then snapshot the
			// playback position, since effectively the rate
			// just changed to 0.
			i.snapshotPosition()
		}
	case Stopped:
		// mpris suggests that Stop should reset the position
		i.lastPosition = 0
		i.lastUpdated = time.Now()
	}
}

func (i *infoReader) updateShuffle() {
	if shuffle, ok := i.Get(mprisShuffle); ok {
		i.Shuffle = shuffle.(bool)
		i.updates.metadata = true
	}
}

func (i *infoReader) updatePosition() {
	if !i.Playing() && !i.Paused() {
		// Some players throw errors when asked for position while stopped.
		return
	}
	if position, ok := i.Get(mprisPosition); ok {
		i.lastUpdated = time.Now()
		i.lastPosition = time.Duration(getLong(position)) * time.Microsecond
		i.updates.position = true
	}
}

func (i *infoReader) updateRate() {
	if rate, ok := i.Get(mprisRate); ok {
		// Position computed based on new rate will not be valid earlier,
		// so snapshot it now for correct values later.
		i.snapshotPosition()
		i.rate = getDouble(rate)
		// Do not mark as updated, will be picked up at next tick.
	}
}

func (i *infoReader) updateMetadata() {
	metadataMap, ok := i.Get(mprisMetadata)
	if !ok {
		return
	}
	metadata := metadataMap.(map[string]dbus.Variant)
	if length, ok := metadata["mpris:length"]; ok {
		i.Length = time.Duration(getLong(length)) * time.Microsecond
		i.updates.metadata = true
	}
	if artist, ok := metadata["xesam:artist"]; ok {
		artists := artist.Value().([]string)
		if len(artists) > 0 {
			i.Artist = artists[0]
			i.updates.metadata = true
		}
	}
	if aArtist, ok := metadata["xesam:albumArtist"]; ok {
		artists := aArtist.Value().([]string)
		if len(artists) > 0 {
			i.AlbumArtist = artists[0]
			i.updates.metadata = true
		}
	}
	if album, ok := metadata["xesam:album"]; ok {
		i.Album = album.Value().(string)
		i.updates.metadata = true
	}
	if title, ok := metadata["xesam:title"]; ok {
		i.Title = title.Value().(string)
		i.updates.metadata = true
	}
	if ArtURL, ok := metadata["mpris:ArtURL"]; ok {
		i.ArtURL = ArtURL.Value().(string)
		i.updates.metadata = true
	}
	if id, ok := metadata["mpris:trackid"]; ok {
		trackID := id.Value().(string)
		if trackID != i.trackID {
			// mpris suggests that position should be reset on track change.
			i.lastPosition = 0
			i.lastUpdated = time.Now()
			i.trackID = trackID
		}
		i.updates.metadata = true
	}
}
