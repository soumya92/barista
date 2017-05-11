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
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
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
	Play()
	Pause()
	PlayPause()
	Stop()
	Next()
	Previous()
	Seek(offset time.Duration)
}

// Config represents a configuration that can be applied to a module instance.
type Config interface {
	apply(*module)
}

// OutputFunc configures a module to display the output of a user-defined function.
type OutputFunc func(Info) bar.Output

func (o OutputFunc) apply(m *module) {
	m.outputFunc = o
}

// OutputTemplate configures a module to display the output of a template.
func OutputTemplate(template func(interface{}) bar.Output) Config {
	return OutputFunc(func(i Info) bar.Output {
		return template(i)
	})
}

// shouldTrackPosition sets whether or not to keep track of the current position.
// Setting this will incur the overhead of also listening to "seeked" signals and
// keeping track of the now playing "rate", but without this calling Position()
// on Info will return garbage.
type shouldTrackPosition bool

func (t shouldTrackPosition) apply(m *module) {
	m.trackPosition = bool(t)
}

// Constant values for whether or not to track the current playback position.
const (
	TrackPosition     = shouldTrackPosition(true)
	DontTrackPosition = shouldTrackPosition(false)
)

// Module is the public interface for a media module.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to control the media player.
type Module interface {
	base.Module
	OnClick(func(Info, Controller, bar.Event))
}

type module struct {
	*base.Base
	playerName    string
	trackPosition bool
	outputFunc    func(Info) bar.Output
	// player state, updated from dbus signals.
	info Info
	// To simplify adding/removing matches and querying metadata,
	// store references to bus and player dbus objects.
	player *mprisPlayer
	// An additional update every second while music is playing
	// to keep the position up to date.
	positionScheduler base.Scheduler
}

// New constructs an instance of the media module with the provided configuration.
func New(player string, config ...Config) Module {
	m := &module{
		Base:       base.New(),
		playerName: player,
	}
	// Set default click handler in New(), can be overridden later.
	m.OnClick(defaultClickHandler)
	// Apply each configuration.
	for _, c := range config {
		c.apply(m)
	}
	// Default output template, if no template/function was specified.
	if m.outputFunc == nil {
		// Construct a simple template that's just the currently playing track.
		defTpl := outputs.TextTemplate(`{{if .Connected}}{{.Title}}{{end}}`)
		OutputTemplate(defTpl).apply(m)
	}
	return m
}

// OnClick sets a click handler for the module.
func (m *module) OnClick(f func(Info, Controller, bar.Event)) {
	if f == nil {
		m.Base.OnClick(nil)
		return
	}
	m.Base.OnClick(func(e bar.Event) {
		f(m.info, m.player, e)
	})
}

// defaultClickHandler provides useful behaviour out of the box,
// Click to play/pause, scroll to seek, and back/forward to switch tracks.
func defaultClickHandler(i Info, c Controller, e bar.Event) {
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
	// Need to handle auth and handshake ourselves for private sessions busses.
	if err := sessionBus.Auth(nil); m.Error(err) {
		return
	}
	if err := sessionBus.Hello(); m.Error(err) {
		return
	}
	m.player = m.newMprisPlayer(sessionBus)
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

// runWithPosition updates the bar output every second,
// in addition to updating it on every signal.
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

// update updates the bar output.
func (m *module) update() {
	m.Output(m.outputFunc(m.info))
	m.positionScheduler.Stop()
	if m.info.PlaybackStatus == Playing {
		m.positionScheduler = m.UpdateEvery(time.Second)
	}
}

// name represents a dbus name that can be decomposed into an interface + member.
type name struct {
	iface  string
	member string
}

func (n name) String() string {
	return n.iface + "." + n.member
}

// buildMatchString builds a match string for the dbus (Add|Remove)Match methods.
func (n name) buildMatchString(sender string, args ...string) string {
	conditions := make([]string, 0)
	conditions = append(conditions, "type='signal'")
	conditions = append(conditions, fmt.Sprintf("interface='%s'", n.iface))
	conditions = append(conditions, fmt.Sprintf("member='%s'", n.member))
	if sender != "" {
		conditions = append(conditions, fmt.Sprintf("sender='%s'", sender))
	}
	for idx, val := range args {
		conditions = append(conditions, fmt.Sprintf("arg%d='%s'", idx, val))
	}
	return strings.Join(conditions, ",")
}

// Constants, signals and properties.
const (
	mprisInterface = "org.mpris.MediaPlayer2.Player"
	dbusInterface  = "org.freedesktop.DBus"
)

// Go doesn't support const structs.
var (
	// dbus methods
	methodNameHasOwner = name{dbusInterface, "NameHasOwner"}
	methodGetNameOwner = name{dbusInterface, "GetNameOwner"}
	methodAddMatch     = name{dbusInterface, "AddMatch"}
	methodRemoveMatch  = name{dbusInterface, "RemoveMatch"}

	// mpris methods
	mprisPlay      = name{mprisInterface, "Play"}
	mprisPause     = name{mprisInterface, "Pause"}
	mprisPlayPause = name{mprisInterface, "PlayPause"}
	mprisStop      = name{mprisInterface, "Stop"}
	mprisNext      = name{mprisInterface, "Next"}
	mprisPrev      = name{mprisInterface, "Previous"}
	mprisSeek      = name{mprisInterface, "Seek"}

	// mpris properties
	mprisRate     = name{mprisInterface, "Rate"}
	mprisPosition = name{mprisInterface, "Position"}
	mprisShuffle  = name{mprisInterface, "Shuffle"}
	mprisStatus   = name{mprisInterface, "PlaybackStatus"}
	mprisMetadata = name{mprisInterface, "Metadata"}

	// Dbus signals used for receiving updates about the media player.
	signalSeeked           = name{mprisInterface, "Seeked"}
	signalNameOwnerChanged = name{dbusInterface, "NameOwnerChanged"}
	signalPropChanged      = name{"org.freedesktop.DBus.Properties", "PropertiesChanged"}
)

// mprisPlayer stores the dbus, player object, and an error together,
// to simplify checking for errors after each call.
type mprisPlayer struct {
	bus           dbus.BusObject
	player        dbus.BusObject
	info          *Info
	trackPosition bool
	err           error
}

func (m *module) newMprisPlayer(sessionBus *dbus.Conn) *mprisPlayer {
	// dbus name for the player.
	dest := fmt.Sprintf("org.mpris.MediaPlayer2.%s", m.playerName)
	// Get the dbus objects for the session bus and the player.
	player := &mprisPlayer{
		player:        sessionBus.Object(dest, "/org/mpris/MediaPlayer2"),
		bus:           sessionBus.BusObject(),
		trackPosition: m.trackPosition,
		info:          &m.info,
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

// Play resumes playback of the current track.
func (m *mprisPlayer) Play() {
	m.Call(mprisPlay)
}

// Pause pauses the track. No effect if not playing.
func (m *mprisPlayer) Pause() {
	m.Call(mprisPause)
}

// PlayPause toggles between play and pause on the media player.
func (m *mprisPlayer) PlayPause() {
	m.Call(mprisPlayPause)
}

// Stop stops and clears the currently playing track.
func (m *mprisPlayer) Stop() {
	m.Call(mprisStop)
}

// Next switches to the next track
func (m *mprisPlayer) Next() {
	m.Call(mprisNext)
}

// Previous switches to the previous track (or restarts the current track).
// Implementation is player-dependent.
func (m *mprisPlayer) Previous() {
	m.Call(mprisPrev)
}

// Seek seeks to the specified offset from the current position.
// Use negative durations to seek backwards.
func (m *mprisPlayer) Seek(offset time.Duration) {
	micros := int64(offset) / int64(time.Microsecond)
	m.Call(mprisSeek, micros)
}

// Get gets a player property from the mpris player object.
func (m *mprisPlayer) Get(prop name) (interface{}, bool) {
	if m.err == nil {
		var v dbus.Variant
		v, m.err = m.player.GetProperty(prop.String())
		if m.err == nil {
			return v.Value(), true
		}
	}
	return nil, false
}

// Call forwards a method call to either the bus or the player as appropriate.
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
	d := m.infoReader(m)
	d.updatePlaybackStatus()
	d.updateMetadata()
	d.updateShuffle()
	d.updatePosition()
	d.updateRate()
}

func (m *mprisPlayer) addMatches(sender string) {
	// Only listen for seek events if the user has indicated that they care about
	// tracking position position.
	if m.trackPosition {
		m.Call(methodAddMatch, signalSeeked.buildMatchString(sender))
	}
	m.Call(methodAddMatch, signalPropChanged.buildMatchString(sender, mprisInterface))
}

func (m *mprisPlayer) removeMatches(sender string) {
	if m.trackPosition {
		m.Call(methodRemoveMatch, signalSeeked.buildMatchString(sender))
	}
	m.Call(methodRemoveMatch, signalPropChanged.buildMatchString(sender, mprisInterface))
}

func (m *mprisPlayer) handleDbusSignal(v *dbus.Signal) (bool, error) {
	switch v.Name {
	case signalPropChanged.String():
		d := m.infoReader(dbusMap(v.Body[1].(map[string]dbus.Variant)))
		d.updatePlaybackStatus()
		d.updateMetadata()
		d.updateShuffle()
		d.updatePosition()
		d.updateRate()
		return d.updated, m.err

	case signalSeeked.String():
		d := m.infoReader(dbusPositionGetter{v})
		d.updatePosition()
		return d.updated, m.err

	case signalNameOwnerChanged.String():
		oldName := v.Body[1].(string)
		newName := v.Body[2].(string)
		if len(oldName) > 0 {
			m.removeMatches(oldName)
		} else {
			// Clear cached info on new name acquisition to play nice with
			// google-play-music-desktop-player.
			*m.info = Info{}
		}
		if len(newName) > 0 {
			m.addMatches(newName)
			m.getInitialInfo()
		} else {
			// Player disconnected from dbus.
			m.info.PlaybackStatus = Disconnected
			return true, m.err
		}
	}
	return false, m.err
}

func (m *mprisPlayer) infoReader(getter dbusGetter) dbusInfoReader {
	return dbusInfoReader{
		Info:          m.info,
		trackPosition: m.trackPosition,
		getter:        getter,
	}
}

// dbusGetter allows getting named dbus properties.
type dbusGetter interface {
	Get(name) (interface{}, bool)
}

// dbusMap implements dbusGetter for a map of string -> variants.
type dbusMap map[string]dbus.Variant

func (d dbusMap) Get(n name) (interface{}, bool) {
	if v, ok := d[n.member]; ok {
		return v.Value(), true
	}
	return nil, false
}

// dbusPositionGetter allows getting a position from a "Seeked" signal
type dbusPositionGetter struct {
	*dbus.Signal
}

func (d dbusPositionGetter) Get(n name) (interface{}, bool) {
	if n == mprisPosition {
		return d.Signal.Body[0], true
	}
	return nil, false
}

// dbusInfoReader providers helper methods to update media info
// from dbus signals or properties.
type dbusInfoReader struct {
	*Info
	getter        dbusGetter
	trackPosition bool
	updated       bool
}

func (d *dbusInfoReader) updatePlaybackStatus() {
	if status, ok := d.getter.Get(mprisStatus); ok {
		oldState := d.PlaybackStatus
		d.PlaybackStatus = PlaybackStatus(status.(string))
		d.updated = true
		if !d.trackPosition {
			// If we're not tracking position, none of the following matters.
			return
		}
		switch d.PlaybackStatus {
		case Playing:
			if oldState == Paused {
				// If we resumed playing, keep current position but mark it as just updated.
				d.lastUpdated = time.Now()
			}
		case Paused:
			if oldState == Playing {
				// If we paused from playing, then snapshot the playback position,
				// since effectively the rate just changed to 0.
				d.snapshotPosition()
			}
		case Stopped:
			// mpris suggests that Stop should reset the position
			d.lastPosition = 0
			d.lastUpdated = time.Now()
		}
	}
}

func (d *dbusInfoReader) updateShuffle() {
	if shuffle, ok := d.getter.Get(mprisShuffle); ok {
		d.Shuffle = shuffle.(bool)
		d.updated = true
	}
}

func (d *dbusInfoReader) updatePosition() {
	if !d.trackPosition {
		return
	}
	if !d.Playing() && !d.Paused() {
		// Some players throw errors when asked for position while stopped.
		return
	}
	if position, ok := d.getter.Get(mprisPosition); ok {
		d.lastUpdated = time.Now()
		d.lastPosition = time.Duration(getLong(position)) * time.Microsecond
		d.updated = true
	}
}

func (d *dbusInfoReader) updateRate() {
	if !d.trackPosition {
		return
	}
	if rate, ok := d.getter.Get(mprisRate); ok {
		// Position computed based on new rate will not be valid earlier,
		// so snapshot it now for correct values later.
		d.snapshotPosition()
		d.rate = getDouble(rate)
		// Do not mark as updated, will be picked up at next tick.
	}
}

func (d *dbusInfoReader) updateMetadata() {
	metadataMap, ok := d.getter.Get(mprisMetadata)
	if !ok {
		return
	}
	metadata := metadataMap.(map[string]dbus.Variant)
	if length, ok := metadata["mpris:length"]; ok {
		d.Length = time.Duration(getLong(length)) * time.Microsecond
		d.updated = true
	}
	if artist, ok := metadata["xesam:artist"]; ok {
		artists := artist.Value().([]string)
		if len(artists) > 0 {
			d.Artist = artists[0]
			d.updated = true
		}
	}
	if aArtist, ok := metadata["xesam:albumArtist"]; ok {
		artists := aArtist.Value().([]string)
		if len(artists) > 0 {
			d.AlbumArtist = artists[0]
			d.updated = true
		}
	}
	if album, ok := metadata["xesam:album"]; ok {
		d.Album = album.Value().(string)
		d.updated = true
	}
	if title, ok := metadata["xesam:title"]; ok {
		d.Title = title.Value().(string)
		d.updated = true
	}
	if ArtURL, ok := metadata["mpris:ArtURL"]; ok {
		d.ArtURL = ArtURL.Value().(string)
		d.updated = true
	}
	if !d.trackPosition {
		// Don't bother setting track id if position doesn't matter.
		return
	}
	if id, ok := metadata["mpris:trackid"]; ok {
		trackID := id.Value().(string)
		if trackID != d.trackID {
			// mpris suggests that position should be reset on track change.
			d.lastPosition = 0
			d.lastUpdated = time.Now()
			d.trackID = trackID
		}
	}
}

// Some mpris players report numeric values as the wrong type. Fix that.
// TODO: See if this is a solved problem.

func getLong(l interface{}) int64 {
	switch l := l.(type) {
	case int64:
		return l
	case int:
		return int64(l)
	case int8:
		return int64(l)
	case int16:
		return int64(l)
	case int32:
		return int64(l)
	case uint:
		return int64(l)
	case uint8:
		return int64(l)
	case uint16:
		return int64(l)
	case uint32:
		return int64(l)
	case uint64:
		return int64(l)
	case float32:
		return int64(l)
	case float64:
		return int64(l)
	case dbus.Variant:
		return getLong(l.Value())
	default:
		return 0
	}
}

func getDouble(d interface{}) float64 {
	switch d := d.(type) {
	case float64:
		return d
	case int:
		return float64(d)
	case int8:
		return float64(d)
	case int16:
		return float64(d)
	case int32:
		return float64(d)
	case int64:
		return float64(d)
	case uint:
		return float64(d)
	case uint8:
		return float64(d)
	case uint16:
		return float64(d)
	case uint32:
		return float64(d)
	case uint64:
		return float64(d)
	case float32:
		return float64(d)
	case dbus.Variant:
		return getDouble(d.Value())
	default:
		return 0.0
	}
}
