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
	"reflect"
	"strings"
	"time"

	"barista.run/timing"

	"github.com/godbus/dbus/v5"
)

// Paused returns true if the media player is connected and paused.
func (i Info) Paused() bool { return i.PlaybackStatus == Paused }

// Playing returns true if the media player is connected and playing media.
func (i Info) Playing() bool { return i.PlaybackStatus == Playing }

// Stopped returns true if the media player is connected but stopped.
func (i Info) Stopped() bool { return i.PlaybackStatus == Stopped }

// Connected returns true if the media player is connected.
func (i Info) Connected() bool { return i.PlaybackStatus != Disconnected }

// Position computes the current track position
// based on the last update from the media player.
func (i Info) Position() time.Duration {
	if i.PlaybackStatus == Paused {
		// If paused, then the position is not advancing.
		return i.lastPosition
	}
	elapsed := timing.Now().Sub(i.lastUpdated)
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

// Play starts playback of the current track.
func (i Info) Play() { i.call("Play") }

// Pause pauses the current track, keeping the current playback position.
func (i Info) Pause() { i.call("Pause") }

// PlayPause toggles between playing and paused.
func (i Info) PlayPause() { i.call("PlayPause") }

// Stop stops playback and resets the playback position.
func (i Info) Stop() { i.call("Stop") }

// Next switches to the next track.
func (i Info) Next() { i.call("Next") }

// Previous switches to the previous track.
func (i Info) Previous() { i.call("Previous") }

// Seek seeks to the given position within the currently playing track.
func (i Info) Seek(offset time.Duration) {
	i.call("Seek", int64(offset/time.Microsecond))
}

func (i *Info) set(key string, value interface{}) {
	switch key {
	case "Rate":
		i.snapshotPosition()
		i.rate = getDouble(value)
	case "Position":
		i.lastUpdated = timing.Now()
		i.lastPosition = time.Duration(getDouble(value)) * time.Microsecond
	case "Shuffle":
		i.Shuffle, _ = value.(bool)
	case "PlaybackStatus":
		status, _ := value.(string)
		i.updatePlaybackStatus(status)
	case "Metadata":
		metadata, _ := value.(map[string]dbus.Variant)
		i.updateMetadata(metadata)
	}
}

// snapshotPosition snapshots the playback position,
// useful when updates to rate or playback status would yield incorrect results.
func (i *Info) snapshotPosition() {
	now := timing.Now()
	elapsed := now.Sub(i.lastUpdated)
	i.lastPosition += time.Duration(float64(elapsed) * i.rate)
	i.lastUpdated = now
}

func (i *Info) updatePlaybackStatus(status string) {
	oldState := i.PlaybackStatus
	i.PlaybackStatus = PlaybackStatus(status)
	if i.PlaybackStatus == oldState {
		return
	}
	switch i.PlaybackStatus {
	case Playing:
		if oldState == Paused {
			// If we resumed playing, keep current position
			// but mark it as just updated.
			i.lastUpdated = timing.Now()
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
		i.lastUpdated = timing.Now()
	}
}

func (i *Info) updateMetadata(metadata map[string]dbus.Variant) {
	i.Length = 0
	if length, ok := metadata["mpris:length"]; ok {
		i.Length = time.Duration(getLong(length)) * time.Microsecond
	}
	i.Artist = ""
	if artist, ok := metadata["xesam:artist"]; ok {
		artists, _ := artist.Value().([]string)
		if len(artists) > 0 {
			i.Artist = artists[0]
		}
	}
	i.AlbumArtist = ""
	if aArtist, ok := metadata["xesam:albumArtist"]; ok {
		artists, _ := aArtist.Value().([]string)
		if len(artists) > 0 {
			i.AlbumArtist = artists[0]
		}
	}
	i.Album = ""
	if album, ok := metadata["xesam:album"]; ok {
		i.Album = album.Value().(string)
	}
	i.Title = ""
	if title, ok := metadata["xesam:title"]; ok {
		i.Title = title.Value().(string)
	}
	i.ArtURL = ""
	if ArtURL, ok := metadata["mpris:ArtURL"]; ok {
		i.ArtURL = ArtURL.Value().(string)
	}
	trackID := ""
	if id, ok := metadata["mpris:trackid"]; ok {
		trackID = id.String()
	}
	if trackID != i.trackID {
		// mpris suggests that position should be reset on track change.
		i.lastPosition = 0
		i.lastUpdated = timing.Now()
		i.trackID = trackID
	}
}

// Some mpris players report numeric values as the wrong type. Fix that.
// TODO: See if this is a solved problem.

func getLong(l interface{}) int64 {
	switch l.(type) {
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(l).Int()
	case uint, uint8, uint16, uint32, uint64:
		return int64(reflect.ValueOf(l).Uint())
	case float32, float64:
		return int64(reflect.ValueOf(l).Float())
	case dbus.Variant:
		return getLong(l.(dbus.Variant).Value())
	default:
		return 0
	}
}

func getDouble(d interface{}) float64 {
	switch d.(type) {
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(d).Int())
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(d).Uint())
	case float32, float64:
		return reflect.ValueOf(d).Float()
	case dbus.Variant:
		return getDouble(d.(dbus.Variant).Value())
	default:
		return 0.0
	}
}
