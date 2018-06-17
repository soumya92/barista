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

	"github.com/godbus/dbus"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/outputs"
)

// Volume represents the current audio volume and mute state.
type Volume struct {
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

	// Infinite loop - wait for updates, and call the update function
	// with new volumes. In the event of an error, return it.
	worker(update func(Volume)) error
}

// Module represents a bar.Module that displays alsa volume information.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to control the system volume, and the
// usual output formatting options.
type Module struct {
	outputFunc    base.Value      // of func(Volume) bar.Output
	clickHandler  base.Value      // of func(Volume, Controller, bar.Event)
	currentVolume base.ErrorValue // of Volume
	impl          moduleImpl
}

// Common parts between all implementations.

// OutputFunc configures a module to display the output of a user-defined
// function.
func (m *Module) OutputFunc(outputFunc func(Volume) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// OutputTemplate configures a module to display the output of a template.
func (m *Module) OutputTemplate(template func(interface{}) bar.Output) *Module {
	return m.OutputFunc(func(v Volume) bar.Output {
		return template(v)
	})
}

// OnClick sets the click handler for the module.
func (m *Module) OnClick(f func(Volume, Controller, bar.Event)) {
	if f == nil {
		f = func(v Volume, c Controller, e bar.Event) {}
	}
	m.clickHandler.Set(f)
}

// Click handles click events on the module's output.
func (m *Module) Click(e bar.Event) {
	handler := m.clickHandler.Get().(func(Volume, Controller, bar.Event))
	if vol, _ := m.currentVolume.Get(); vol != nil {
		handler(vol.(Volume), m, e)
	}
}

// Throttle volume updates to a ~20ms to prevent alsa breakage.
var lastVolumeChangeTime time.Time

// DefaultClickHandler provides a simple example of the click handler capabilities.
// It toggles mute on left click, and raises/lowers the volume on scroll.
func DefaultClickHandler(v Volume, c Controller, e bar.Event) {
	now := time.Now()
	if lastVolumeChangeTime.Add(20 * time.Millisecond).After(now) {
		// Don't update the volume if it was updated <20ms ago.
		return
	}
	lastVolumeChangeTime = now
	if e.Button == bar.ButtonLeft {
		c.SetMuted(!v.Mute)
		return
	}
	volStep := (v.Max - v.Min) / 100
	if volStep == 0 {
		volStep = 1
	}
	if e.Button == bar.ScrollUp {
		c.SetVolume(v.Vol + volStep)
	}
	if e.Button == bar.ScrollDown {
		c.SetVolume(v.Vol - volStep)
	}
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	go m.runWorker()
	v, err := m.currentVolume.Get()
	outputFunc := m.outputFunc.Get().(func(Volume) bar.Output)
	for {
		if s.Error(err) {
			return
		}
		if vol, ok := v.(Volume); ok {
			s.Output(outputFunc(vol))
		}
		select {
		case <-m.currentVolume.Update():
			v, err = m.currentVolume.Get()
		case <-m.outputFunc.Update():
			outputFunc = m.outputFunc.Get().(func(Volume) bar.Output)
		}
	}
}

// SetVolume tells the implementation to update the volume, and
// updates the stored volume if the implementation-specific call
// was successful.
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

// SetVolume tells the implementation to update the mute state, and
// updates the stored volume if the implementation-specific call
// was successful.
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

// init initializes the Module struct to usable default values.
func createModule(impl moduleImpl) *Module {
	m := &Module{
		impl: impl,
	}

	l.Register(m, "outputFunc", "currentVolume", "clickHandler")
	m.OnClick(DefaultClickHandler)
	// Default output template is just the volume %, "MUT" when muted.
	m.OutputTemplate(outputs.TextTemplate(`{{if .Mute}}MUT{{else}}{{.Pct}}%{{end}}`))

	return m
}

// runWorker runs the worker from the moduleImpl interface, and sets
// the error on currentVolume if an error results.
func (m *Module) runWorker() {
	err := m.impl.worker(func(v Volume) { m.currentVolume.Set(v) })
	m.currentVolume.Error(err)
}

// ALSA implementation.

type alsaModule struct {
	cardName  string
	mixerName string

	// To make it easier to change volume using alsa apis,
	// store the current state and snd_mixer_elem_t pointer.
	elem *C.snd_mixer_elem_t
}

// Mixer constructs an instance of the volume module for a
// specific card and mixer on that card.
func Mixer(card, mixer string) *Module {
	impl := &alsaModule{
		cardName:  card,
		mixerName: mixer,
	}
	m := createModule(impl)
	l.Labelf(m, "%s,%s", card, mixer)
	return m
}

// DefaultMixer constructs an instance of the volume module for the default mixer.
func DefaultMixer() *Module {
	return Mixer("default", "Master")
}

// SetVolume sets the system volume.
// It does not change the mute status.
func (m *alsaModule) setVolume(newVol int64) error {
	C.snd_mixer_selem_set_playback_volume_all(m.elem, C.long(newVol))
	return nil
}

// SetMuted controls whether the system volume is muted.
func (m *alsaModule) setMuted(muted bool) error {
	if muted {
		C.snd_mixer_selem_set_playback_switch_all(m.elem, C.int(0))
	} else {
		C.snd_mixer_selem_set_playback_switch_all(m.elem, C.int(1))
	}
	return nil
}

// worker waits for signals from alsa and updates the stored volume.
func (m *alsaModule) worker(update func(Volume)) error {
	cardName := C.CString(m.cardName)
	defer C.free(unsafe.Pointer(cardName))
	mixerName := C.CString(m.mixerName)
	defer C.free(unsafe.Pointer(mixerName))
	// Structs for querying ALSA.
	var handle *C.snd_mixer_t
	var sid *C.snd_mixer_selem_id_t
	// Set up query for master mixer.
	if err := int(C.snd_mixer_selem_id_malloc(&sid)); err < 0 {
		return fmt.Errorf("snd_mixer_selem_id_malloc: %d", err)
	}
	defer C.snd_mixer_selem_id_free(sid)
	C.snd_mixer_selem_id_set_index(sid, 0)
	C.snd_mixer_selem_id_set_name(sid, mixerName)
	// Connect to alsa
	if err := int(C.snd_mixer_open(&handle, 0)); err < 0 {
		return fmt.Errorf("snd_mixer_open: %d", err)
	}
	defer C.snd_mixer_close(handle)
	if err := int(C.snd_mixer_attach(handle, cardName)); err < 0 {
		return fmt.Errorf("snd_mixer_attach: %d", err)
	}
	defer C.snd_mixer_detach(handle, cardName)
	if err := int(C.snd_mixer_load(handle)); err < 0 {
		return fmt.Errorf("snd_mixer_load: %d", err)
	}
	defer C.snd_mixer_free(handle)
	if err := int(C.snd_mixer_selem_register(handle, nil, nil)); err < 0 {
		return fmt.Errorf("snd_mixer_selem_register: %d", err)
	}
	// Get master default thing
	m.elem = C.snd_mixer_find_selem(handle, sid)
	if m.elem == nil {
		return fmt.Errorf("snd_mixer_find_selem NULL")
	}
	var min, max, vol C.long
	var mute C.int
	C.snd_mixer_selem_get_playback_volume_range(m.elem, &min, &max)
	for {
		C.snd_mixer_selem_get_playback_volume(m.elem, C.SND_MIXER_SCHN_MONO, &vol)
		C.snd_mixer_selem_get_playback_switch(m.elem, C.SND_MIXER_SCHN_MONO, &mute)
		update(Volume{
			Min:  int64(min),
			Max:  int64(max),
			Vol:  int64(vol),
			Mute: (int(mute) == 0),
		})
		if err := int(C.snd_mixer_wait(handle, -1)); err < 0 {
			return fmt.Errorf("snd_mixer_wait: %d", err)
		}
		if err := int(C.snd_mixer_handle_events(handle)); err < 0 {
			return fmt.Errorf("snd_mixer_handle_events: %d", err)
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

func createPulseModule(sinkName string) *Module {
	impl := &paModule{
		sinkName: sinkName,
	}
	m := createModule(impl)

	label := sinkName
	if label == "" {
		label = "default"
	}
	l.Labelf(m, "pulse:%s", label)

	return m
}

// Create a PulseAudio volume module that follows the default sink.
func DefaultSink() *Module {
	return createPulseModule("")
}

// Create a PulseAudio volume module for a named sink.
func Sink(sinkName string) *Module {
	return createPulseModule(sinkName)
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
	if err := m.listen("Device.MuteUpdated", sink); err != nil {
		return err
	}

	return nil
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

func (m *paModule) refresh(update func(Volume)) error {
	v := Volume{}
	v.Min = 0

	max, err := m.sink.GetProperty("org.PulseAudio.Core1.Device.BaseVolume")
	if err != nil {
		return err
	}
	v.Max = int64(max.Value().(uint32))

	vol, err := m.sink.GetProperty("org.PulseAudio.Core1.Device.Volume")
	if err != nil {
		return err
	}

	// Take the volume as the average across all channels.
	var totalVol int64
	channels := vol.Value().([]uint32)
	for _, ch := range channels {
		totalVol += int64(ch)
	}
	v.Vol = totalVol / int64(len(channels))

	mute, err := m.sink.GetProperty("org.PulseAudio.Core1.Device.Mute")
	if err != nil {
		return err
	}
	v.Mute = mute.Value().(bool)

	update(v)
	return nil
}

func (m *paModule) worker(update func(Volume)) error {
	conn, err := openPulseAudio()
	if err != nil {
		return err
	}
	m.conn = conn
	defer func() {
		conn.Close()
		m.conn = nil
	}()

	m.core = conn.Object("org.PulseAudio.Core1", "/org/pulseaudio/core1")
	defer func() {
		m.core = nil
	}()

	if m.sinkName != "" {
		if err = m.openSinkByName(m.sinkName); err != nil {
			return err
		}
	} else {
		if err = m.openFallbackSink(); err != nil {
			return err
		}

		if err = m.listen("FallbackSinkUpdated"); err != nil {
			return err
		}
	}

	defer func() {
		m.sink = nil
	}()

	if err = m.refresh(update); err != nil {
		return err
	}

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	// Listen for signals from D-Bus, and update appropriately.
	for signal := range signals {
		// If the fallback sink changed, open the new one.
		if m.sinkName == "" && signal.Path == m.core.Path() {
			if err = m.openFallbackSink(); err != nil {
				return err
			}
		}
		if err = m.refresh(update); err != nil {
			return err
		}
	}

	return nil
}
