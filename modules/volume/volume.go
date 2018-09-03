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

// Package volume provides an i3bar module that interfaces with alsa
// to display and control the system volume.
package volume

/*
  #cgo pkg-config: alsa
  #include <alsa/asoundlib.h>
  #include <alsa/mixer.h>
  #include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"time"
	"unsafe"

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"

	"github.com/godbus/dbus"
	"golang.org/x/time/rate"
)

// Volume represents the current audio volume and mute state.
type Volume struct {
	Controller
	Min, Max, Vol int64
	Mute          bool
}

// Frac returns the current volume as a fraction of the total range.
func (v Volume) Frac() float64 {
	return float64(v.Vol-v.Min) / float64(v.Max-v.Min)
}

// Pct returns the current volume in the range 0-100.
func (v Volume) Pct() int {
	return int((v.Frac() * 100) + 0.5)
}

// Controller provides an interface to change the system volume from the click handler.
type Controller interface {

	// SetMuted controls whether the system volume is muted.
	SetMuted(bool)

	// SetVolume sets the system volume.
	// It does not change the mute status.
	SetVolume(int64)
}

// Interface that must be implemented by individual volume implementations.
type moduleImpl interface {
	setMuted(muted bool) error
	setVolume(volume int64) error

	// Infinite loop: push updates and errors to the provided s.
	worker(s *value.ErrorValue)
}

// Module represents a bar.Module that displays volume information.
type Module struct {
	outputFunc    value.Value      // of func(Volume) bar.Output
	clickHandler  value.Value      // of func(Volume, Controller, bar.Event)
	currentVolume value.ErrorValue // of Volume
	impl          moduleImpl
}

// Output configures a module to display the output of a user-defined
// function.
func (m *Module) Output(outputFunc func(Volume) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Throttle volume updates to once every ~20ms to prevent alsa breakage.
var alsaLimiter = rate.NewLimiter(rate.Every(20*time.Millisecond), 1)

// defaultClickHandler provides a simple example of the click handler capabilities.
// It toggles mute on left click, and raises/lowers the volume on scroll.
func defaultClickHandler(v Volume) func(bar.Event) {
	return func(e bar.Event) {
		if !alsaLimiter.Allow() {
			// Don't update the volume if it was updated <20ms ago.
			return
		}
		if e.Button == bar.ButtonLeft {
			v.SetMuted(!v.Mute)
			return
		}
		volStep := (v.Max - v.Min) / 100
		if volStep == 0 {
			volStep = 1
		}
		if e.Button == bar.ScrollUp {
			v.SetVolume(v.Vol + volStep)
		}
		if e.Button == bar.ScrollDown {
			v.SetVolume(v.Vol - volStep)
		}
	}
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	go m.impl.worker(&m.currentVolume)
	v, err := m.currentVolume.Get()
	outputFunc := m.outputFunc.Get().(func(Volume) bar.Output)
	for {
		if s.Error(err) {
			return
		}
		if vol, ok := v.(Volume); ok {
			vol.Controller = m
			s.Output(outputs.Group(outputFunc(vol)).
				OnClick(defaultClickHandler(vol)))
		}
		select {
		case <-m.currentVolume.Next():
			v, err = m.currentVolume.Get()
		case <-m.outputFunc.Next():
			outputFunc = m.outputFunc.Get().(func(Volume) bar.Output)
		}
	}
}

// SetVolume sets the system volume.
// It does not change the mute status.
func (m *Module) SetVolume(volume int64) {
	vol, _ := m.currentVolume.Get()
	if vol == nil {
		return
	}

	v := vol.(Volume)
	if volume > v.Max {
		volume = v.Max
	}
	if volume < v.Min {
		volume = v.Min
	}
	if volume == v.Vol {
		return
	}

	if err := m.impl.setVolume(volume); err != nil {
		l.Log("Error updating volume: %v", err)
	} else {
		v.Vol = volume
		m.currentVolume.Set(v)
	}
}

// SetMuted controls whether the system volume is muted.
func (m *Module) SetMuted(muted bool) {
	vol, _ := m.currentVolume.Get()
	if vol == nil {
		return
	}

	v := vol.(Volume)
	if v.Mute == muted {
		return
	}

	if err := m.impl.setMuted(muted); err != nil {
		l.Log("Error updating mute state: %v", err)
	} else {
		v.Mute = muted
		m.currentVolume.Set(v)
	}
}

// createModule creates a new module with the given backing implementation.
func createModule(impl moduleImpl) *Module {
	m := &Module{impl: impl}
	l.Register(m, "outputFunc", "currentVolume", "clickHandler", "impl")
	// Default output is just the volume %, "MUT" when muted.
	m.Output(func(v Volume) bar.Output {
		if v.Mute {
			return outputs.Text("MUT")
		}
		return outputs.Textf("%d%%", v.Pct())
	})
	return m
}

// ALSA implementation.
type alsaModule struct {
	cardName  string
	mixerName string

	// To make it easier to change volume using alsa apis,
	// store the current state and snd_mixer_elem_t pointer.
	elem *C.snd_mixer_elem_t
}

func alsaError(result C.int, desc string) error {
	if int(result) < 0 {
		return fmt.Errorf("%s: %d", desc, result)
	}
	return nil
}

// Mixer constructs an instance of the volume module for a
// specific card and mixer on that card.
func Mixer(card, mixer string) *Module {
	m := createModule(&alsaModule{
		cardName:  card,
		mixerName: mixer,
	})
	l.Labelf(m, "alsa:%s,%s", card, mixer)
	return m
}

// DefaultMixer constructs an instance of the volume module for the default mixer.
func DefaultMixer() *Module {
	return Mixer("default", "Master")
}

func (m *alsaModule) setVolume(newVol int64) error {
	return alsaError(
		C.snd_mixer_selem_set_playback_volume_all(m.elem, C.long(newVol)),
		"snd_mixer_selem_set_playback_volume_all")
}

func (m *alsaModule) setMuted(muted bool) error {
	var muteInt C.int
	if muted {
		muteInt = C.int(0)
	} else {
		muteInt = C.int(1)
	}
	return alsaError(
		C.snd_mixer_selem_set_playback_switch_all(m.elem, muteInt),
		"snd_mixer_selem_set_playback_switch_all")
}

// worker waits for signals from alsa and updates the stored volume.
func (m *alsaModule) worker(s *value.ErrorValue) {
	cardName := C.CString(m.cardName)
	defer C.free(unsafe.Pointer(cardName))
	mixerName := C.CString(m.mixerName)
	defer C.free(unsafe.Pointer(mixerName))
	// Structs for querying ALSA.
	var handle *C.snd_mixer_t
	var sid *C.snd_mixer_selem_id_t
	// Shortcut for error handling
	var err = func(result C.int, desc string) bool {
		return s.Error(alsaError(result, desc))
	}
	// Set up query for master mixer.
	if err(C.snd_mixer_selem_id_malloc(&sid), "snd_mixer_selem_id_malloc") {
		return
	}
	defer C.snd_mixer_selem_id_free(sid)
	C.snd_mixer_selem_id_set_index(sid, 0)
	C.snd_mixer_selem_id_set_name(sid, mixerName)
	// Connect to alsa
	if err(C.snd_mixer_open(&handle, 0), "snd_mixer_open") {
		return
	}
	defer C.snd_mixer_close(handle)
	if err(C.snd_mixer_attach(handle, cardName), "snd_mixer_attach") {
		return
	}
	defer C.snd_mixer_detach(handle, cardName)
	if err(C.snd_mixer_load(handle), "snd_mixer_load") {
		return
	}
	defer C.snd_mixer_free(handle)
	if err(C.snd_mixer_selem_register(handle, nil, nil), "snd_mixer_selem_register") {
		return
	}
	// Get master default thing
	m.elem = C.snd_mixer_find_selem(handle, sid)
	if m.elem == nil {
		s.Error(fmt.Errorf("snd_mixer_find_selem NULL"))
		return
	}
	var min, max, vol C.long
	var mute C.int
	C.snd_mixer_selem_get_playback_volume_range(m.elem, &min, &max)
	for {
		C.snd_mixer_selem_get_playback_volume(m.elem, C.SND_MIXER_SCHN_MONO, &vol)
		C.snd_mixer_selem_get_playback_switch(m.elem, C.SND_MIXER_SCHN_MONO, &mute)
		s.Set(Volume{
			Min:  int64(min),
			Max:  int64(max),
			Vol:  int64(vol),
			Mute: (int(mute) == 0),
		})
		if err(C.snd_mixer_wait(handle, -1), "snd_mixer_wait") {
			return
		}
		if err(C.snd_mixer_handle_events(handle), "snd_mixer_handle_events") {
			return
		}
	}
}

// PulseAudio implementation.
type paModule struct {
	conn     *dbus.Conn
	core     dbus.BusObject
	sink     dbus.BusObject
	sinkName string
}

func dialAndAuth(addr string) (*dbus.Conn, error) {
	conn, err := dbus.Dial(addr)
	if err != nil {
		return nil, err
	}
	err = conn.Auth(nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func openPulseAudio() (*dbus.Conn, error) {
	// Pulse defaults to creating its socket in a well-known place under
	// XDG_RUNTIME_DIR. For Pulse instances created by systemd, this is the
	// only reliable way to contact Pulse via D-Bus, since Pulse is created
	// on a per-user basis, but the session bus is created once for every
	// session, and a user can have multiple sessions.
	xdgDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgDir != "" {
		addr := fmt.Sprintf("unix:path=%s/pulse/dbus-socket", xdgDir)
		return dialAndAuth(addr)
	}

	// Couldn't find the PulseAudio bus on the fast path, so look for it
	// by querying the session bus.
	bus, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, err
	}
	defer bus.Close()
	err = bus.Auth(nil)
	if err != nil {
		return nil, err
	}

	locator := bus.Object("org.PulseAudio1", "/org/pulseaudio/server_lookup1")
	path, err := locator.GetProperty("org.PulseAudio.ServerLookup1.Address")
	if err != nil {
		return nil, err
	}

	return dialAndAuth(path.Value().(string))
}

// Sink creates a PulseAudio volume module for a named sink.
func Sink(sinkName string) *Module {
	m := createModule(&paModule{sinkName: sinkName})
	if sinkName == "" {
		sinkName = "default"
	}
	l.Labelf(m, "pulse:%s", sinkName)
	return m
}

// DefaultSink creates a PulseAudio volume module that follows the default sink.
func DefaultSink() *Module {
	return Sink("")
}

func (m *paModule) setVolume(newVol int64) error {
	if m.sink == nil {
		return fmt.Errorf("Sink not ready")
	}

	call := m.sink.Call("org.freedesktop.DBus.Properties.Set", 0,
		"org.PulseAudio.Core1.Device", "Volume", dbus.MakeVariant([]uint32{uint32(newVol)}))
	return call.Err
}

func (m *paModule) setMuted(muted bool) error {
	if m.sink == nil {
		return fmt.Errorf("Sink not ready")
	}

	call := m.sink.Call("org.freedesktop.DBus.Properties.Set", 0,
		"org.PulseAudio.Core1.Device", "Mute", dbus.MakeVariant(muted))
	return call.Err
}

func (m *paModule) listen(signal string, objects ...dbus.ObjectPath) error {
	call := m.core.Call("org.PulseAudio.Core1.ListenForSignal", 0, "org.PulseAudio.Core1."+signal, objects)
	return call.Err
}

func (m *paModule) openSink(sink dbus.ObjectPath) error {
	m.sink = m.conn.Object("org.PulseAudio.Core1.Sink", sink)
	if err := m.listen("Device.VolumeUpdated", sink); err != nil {
		return err
	}
	return m.listen("Device.MuteUpdated", sink)
}

func (m *paModule) openSinkByName(name string) error {
	var path dbus.ObjectPath
	err := m.core.Call("org.PulseAudio.Core1.GetSinkByName", 0, name).Store(&path)
	if err != nil {
		return err
	}
	return m.openSink(path)
}

func (m *paModule) openFallbackSink() error {
	path, err := m.core.GetProperty("org.PulseAudio.Core1.FallbackSink")
	if err != nil {
		return err
	}
	return m.openSink(path.Value().(dbus.ObjectPath))
}

func (m *paModule) updateVolume(s *value.ErrorValue) {
	v := Volume{}
	v.Min = 0

	max, err := m.sink.GetProperty("org.PulseAudio.Core1.Device.BaseVolume")
	if s.Error(err) {
		return
	}
	v.Max = int64(max.Value().(uint32))

	vol, err := m.sink.GetProperty("org.PulseAudio.Core1.Device.Volume")
	if s.Error(err) {
		return
	}

	// Take the volume as the average across all channels.
	var totalVol int64
	channels := vol.Value().([]uint32)
	for _, ch := range channels {
		totalVol += int64(ch)
	}
	v.Vol = totalVol / int64(len(channels))

	mute, err := m.sink.GetProperty("org.PulseAudio.Core1.Device.Mute")
	if s.Error(err) {
		return
	}
	v.Mute = mute.Value().(bool)
	s.Set(v)
}

func (m *paModule) worker(s *value.ErrorValue) {
	conn, err := openPulseAudio()
	if s.Error(err) {
		return
	}
	m.conn = conn
	defer func() {
		conn.Close()
		m.conn = nil
	}()

	m.core = conn.Object("org.PulseAudio.Core1", "/org/pulseaudio/core1")
	defer func() { m.core = nil }()

	if m.sinkName != "" {
		if s.Error(m.openSinkByName(m.sinkName)) {
			return
		}
	} else {
		if s.Error(m.openFallbackSink()) {
			return
		}
		if s.Error(m.listen("FallbackSinkUpdated")) {
			return
		}
	}
	defer func() { m.sink = nil }()

	m.updateVolume(s)

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	// Listen for signals from D-Bus, and update appropriately.
	for signal := range signals {
		// If the fallback sink changed, open the new one.
		if m.sinkName == "" && signal.Path == m.core.Path() {
			if s.Error(m.openFallbackSink()) {
				return
			}
		}
		m.updateVolume(s)
	}
}
