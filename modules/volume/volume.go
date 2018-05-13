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
	"time"
	"unsafe"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
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
	return int(v.Frac() * 100)
}

// Controller provides an interface to change the system volume from the click handler.
type Controller interface {

	// SetMuted controls whether the system volume is muted.
	SetMuted(bool)

	// SetVolume sets the system volume.
	// It does not change the mute status.
	SetVolume(int64)
}

// Module represents a bar.Module that displays alsa volume information.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to control the system volume, and the
// usual output formatting options.
type Module struct {
	cardName  string
	mixerName string

	outputFunc    base.Value      // of func(Volume) bar.Output
	clickHandler  base.Value      // of func(Volume, Controller, bar.Event)
	currentVolume base.ErrorValue // of Volume

	// To make it easier to change volume using alsa apis,
	// store the current state and snd_mixer_elem_t pointer.
	elem *C.snd_mixer_elem_t
}

// Mixer constructs an instance of the volume module for a
// specific card and mixer on that card.
func Mixer(card, mixer string) *Module {
	m := &Module{cardName: card, mixerName: mixer}
	m.OnClick(DefaultClickHandler)
	// Default output template is just the volume %, "MUT" when muted.
	m.OutputTemplate(outputs.TextTemplate(`{{if .Mute}}MUT{{else}}{{.Pct}}%{{end}}`))
	return m
}

// DefaultMixer constructs an instance of the volume module for the default mixer.
func DefaultMixer() *Module {
	return Mixer("default", "Master")
}

// SetVolume sets the system volume.
// It does not change the mute status.
func (m *Module) SetVolume(newVol int64) {
	vol, _ := m.currentVolume.Get()
	if vol == nil {
		return
	}
	v := vol.(Volume)
	if newVol > v.Max {
		newVol = v.Max
	}
	if newVol < v.Min {
		newVol = v.Min
	}
	v.Vol = newVol
	C.snd_mixer_selem_set_playback_volume_all(m.elem, C.long(v.Vol))
	m.currentVolume.Set(v)
}

// SetMuted controls whether the system volume is muted.
func (m *Module) SetMuted(muted bool) {
	vol, _ := m.currentVolume.Get()
	if vol == nil {
		return
	}
	v := vol.(Volume)
	if muted {
		C.snd_mixer_selem_set_playback_switch_all(m.elem, C.int(0))
	} else {
		C.snd_mixer_selem_set_playback_switch_all(m.elem, C.int(1))
	}
	v.Mute = muted
	m.currentVolume.Set(v)
}

// OutputFunc configures a module to display the output of a user-defined function.
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
func (m *Module) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker()
	go m.outputLoop(ch)
	return ch
}

// worker waits for signals from alsa and updates the stored volume.
func (m *Module) worker() {
	cardName := C.CString(m.cardName)
	defer C.free(unsafe.Pointer(cardName))
	mixerName := C.CString(m.mixerName)
	defer C.free(unsafe.Pointer(mixerName))
	// Structs for querying ALSA.
	var handle *C.snd_mixer_t
	var sid *C.snd_mixer_selem_id_t
	// Set up query for master mixer.
	if err := int(C.snd_mixer_selem_id_malloc(&sid)); err < 0 {
		m.currentVolume.Error(fmt.Errorf("snd_mixer_selem_id_malloc: %d", err))
		return
	}
	C.snd_mixer_selem_id_set_index(sid, 0)
	C.snd_mixer_selem_id_set_name(sid, mixerName)
	// Connect to alsa
	if err := int(C.snd_mixer_open(&handle, 0)); err < 0 {
		m.currentVolume.Error(fmt.Errorf("snd_mixer_open: %d", err))
		return
	}
	if err := int(C.snd_mixer_attach(handle, cardName)); err < 0 {
		m.currentVolume.Error(fmt.Errorf("snd_mixer_attach: %d", err))
		return
	}
	if err := int(C.snd_mixer_load(handle)); err < 0 {
		m.currentVolume.Error(fmt.Errorf("snd_mixer_load: %d", err))
		return
	}
	if err := int(C.snd_mixer_selem_register(handle, nil, nil)); err < 0 {
		m.currentVolume.Error(fmt.Errorf("snd_mixer_selem_register: %d", err))
		return
	}
	// Get master default thing
	m.elem = C.snd_mixer_find_selem(handle, sid)
	if m.elem == nil {
		m.currentVolume.Error(fmt.Errorf("snd_mixer_find_selem NULL"))
		return
	}
	var min, max, vol C.long
	var mute C.int
	C.snd_mixer_selem_get_playback_volume_range(m.elem, &min, &max)
	for {
		C.snd_mixer_selem_get_playback_volume(m.elem, C.SND_MIXER_SCHN_MONO, &vol)
		C.snd_mixer_selem_get_playback_switch(m.elem, C.SND_MIXER_SCHN_MONO, &mute)
		m.currentVolume.Set(Volume{
			Min:  int64(min),
			Max:  int64(max),
			Vol:  int64(vol),
			Mute: (int(mute) == 0),
		})
		if err := int(C.snd_mixer_wait(handle, -1)); err < 0 {
			m.currentVolume.Error(fmt.Errorf("snd_mixer_wait: %d", err))
			return
		}
		if err := int(C.snd_mixer_handle_events(handle)); err < 0 {
			m.currentVolume.Error(fmt.Errorf("snd_mixer_handle_events: %d", err))
			return
		}
	}
}

// outputLoop listens for updates to the volume, as well as the output function,
// and updates the module output.
func (m *Module) outputLoop(ch base.Channel) {
	v, err := m.currentVolume.Get()
	sVol := m.currentVolume.Subscribe()

	outputFunc := m.outputFunc.Get().(func(Volume) bar.Output)
	sOutputFunc := m.outputFunc.Subscribe()

	for {
		if ch.Error(err) {
			return
		}
		if vol, ok := v.(Volume); ok {
			ch.Output(outputFunc(vol))
		}
		select {
		case <-sVol:
			v, err = m.currentVolume.Get()
		case <-sOutputFunc:
			outputFunc = m.outputFunc.Get().(func(Volume) bar.Output)
		}
	}
}
