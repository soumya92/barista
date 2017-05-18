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
	"github.com/soumya92/barista/modules/base"
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

// Module is the public interface for the volume module.
// In addition to bar.Module, it also provides an expanded OnClick,
// which allows click handlers to control the system volume, and the
// usual output formatting options.
type Module interface {
	base.Module

	// OutputFunc configures a module to display the output of a user-defined function.
	OutputFunc(func(Volume) bar.Output) Module

	// OutputTemplate configures a module to display the output of a template.
	OutputTemplate(func(interface{}) bar.Output) Module

	// OnClick sets the click handler for a module.
	OnClick(func(Volume, Controller, bar.Event))
}

type module struct {
	*base.Base
	cardName   string
	mixerName  string
	outputFunc func(Volume) bar.Output
	// To make it easier to change volume using alsa apis,
	// store the current state and snd_mixer_elem_t pointer.
	elem          *C.snd_mixer_elem_t
	min, max, vol C.long
	mute          C.int
}

// Mixer constructs an instance of the volume module for a
// specific card and mixer on that card.
func Mixer(card, mixer string) Module {
	m := &module{
		Base:      base.New(),
		cardName:  card,
		mixerName: mixer,
	}
	m.OnClick(DefaultClickHandler)
	// Default output template is just the volume %, "MUT" when muted.
	m.OutputTemplate(outputs.TextTemplate(`{{if .Mute}}MUT{{else}}{{.Pct}}%{{end}}`))
	return m
}

// DefaultMixer constructs an instance of the volume module for the default mixer.
func DefaultMixer() Module {
	return Mixer("default", "Master")
}

func (m *module) SetVolume(newVol int64) {
	if max := int64(m.max); newVol > max {
		newVol = max
	}
	if min := int64(m.min); newVol < min {
		newVol = min
	}
	m.vol = C.long(newVol)
	C.snd_mixer_selem_set_playback_volume_all(m.elem, m.vol)
	m.Update()
}

func (m *module) SetMuted(muted bool) {
	if muted {
		m.mute = C.int(0)
	} else {
		m.mute = C.int(1)
	}
	C.snd_mixer_selem_set_playback_switch_all(m.elem, m.mute)
	m.Update()
}

func (m *module) OutputFunc(outputFunc func(Volume) bar.Output) Module {
	m.outputFunc = outputFunc
	m.Update()
	return m
}

func (m *module) OutputTemplate(template func(interface{}) bar.Output) Module {
	return m.OutputFunc(func(v Volume) bar.Output {
		return template(v)
	})
}

func (m *module) OnClick(f func(Volume, Controller, bar.Event)) {
	if f == nil {
		m.Base.OnClick(nil)
		return
	}
	m.Base.OnClick(func(e bar.Event) {
		f(m.volume(), m, e)
	})
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

func (m *module) Stream() <-chan bar.Output {
	// Worker goroutine to update the volume when notified by alsa.
	go func() { m.Error(m.worker()) }()
	return m.Base.Stream()
}

// worker continuously waits for signals from alsa and refreshes
// the module whenever the volume changes.
func (m *module) worker() error {
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
	C.snd_mixer_selem_id_set_index(sid, 0)
	C.snd_mixer_selem_id_set_name(sid, mixerName)
	// Connect to alsa
	if err := int(C.snd_mixer_open(&handle, 0)); err < 0 {
		return fmt.Errorf("snd_mixer_open: %d", err)
	}
	if err := int(C.snd_mixer_attach(handle, cardName)); err < 0 {
		return fmt.Errorf("snd_mixer_attach: %d", err)
	}
	if err := int(C.snd_mixer_load(handle)); err < 0 {
		return fmt.Errorf("snd_mixer_load: %d", err)
	}
	if err := int(C.snd_mixer_selem_register(handle, nil, nil)); err < 0 {
		return fmt.Errorf("snd_mixer_selem_register: %d", err)
	}
	// Get master default thing
	m.elem = C.snd_mixer_find_selem(handle, sid)
	if m.elem == nil {
		return fmt.Errorf("snd_mixer_find_selem NULL")
	}
	C.snd_mixer_selem_get_playback_volume_range(m.elem, &m.min, &m.max)
	m.OnUpdate(m.update)
	for {
		C.snd_mixer_selem_get_playback_volume(m.elem, C.SND_MIXER_SCHN_MONO, &m.vol)
		C.snd_mixer_selem_get_playback_switch(m.elem, C.SND_MIXER_SCHN_MONO, &m.mute)
		m.Update()
		if err := int(C.snd_mixer_wait(handle, -1)); err < 0 {
			return fmt.Errorf("snd_mixer_wait: %d", err)
		}
		if err := int(C.snd_mixer_handle_events(handle)); err < 0 {
			return fmt.Errorf("snd_mixer_handle_events: %d", err)
		}
	}
}

func (m *module) volume() Volume {
	return Volume{
		Min:  int64(m.min),
		Max:  int64(m.max),
		Vol:  int64(m.vol),
		Mute: (int(m.mute) == 0),
	}
}

func (m *module) update() {
	m.Output(m.outputFunc(m.volume()))
}
