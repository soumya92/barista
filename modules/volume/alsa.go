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

package volume

/*
  #cgo pkg-config: alsa
  #include <asoundlib.h>
*/
import "C"
import (
	"fmt"

	"barista.run/base/value"
	l "barista.run/logging"
)

//go:generate ruby capi.rb

// ALSA implementation.
type alsaModule struct {
	cardName  string
	mixerName string
}

type alsaController struct {
	elem *ctyp_snd_mixer_elem_t
}

func alsaError(result int32, desc string) error {
	if result < 0 {
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

func (c alsaController) setVolume(newVol int64) error {
	return alsaError(
		alsa.snd_mixer_selem_set_playback_volume_all(c.elem, newVol),
		"snd_mixer_selem_set_playback_volume_all")
}

func (c alsaController) setMuted(muted bool) error {
	var muteInt int32
	if muted {
		muteInt = 0
	} else {
		muteInt = 1
	}
	return alsaError(
		alsa.snd_mixer_selem_set_playback_switch_all(c.elem, muteInt),
		"snd_mixer_selem_set_playback_switch_all")
}

// worker waits for signals from alsa and updates the stored volume.
func (m *alsaModule) worker(s *value.ErrorValue) {
	// Structs for querying ALSA.
	var handle *ctyp_snd_mixer_t
	var sid *ctyp_snd_mixer_selem_id_t
	// Shortcut for error handling
	var err = func(result int32, desc string) bool {
		return s.Error(alsaError(result, desc))
	}
	if err(alsa.snd_mixer_selem_id_malloc(&sid), "snd_mixer_selem_id_malloc") {
		return
	}
	defer alsa.snd_mixer_selem_id_free(sid)
	alsa.snd_mixer_selem_id_set_index(sid, 0)
	alsa.snd_mixer_selem_id_set_name(sid, m.mixerName)
	// Connect to alsa
	if err(alsa.snd_mixer_open(&handle, 0), "snd_mixer_open") {
		return
	}
	defer alsa.snd_mixer_close(handle)
	if err(alsa.snd_mixer_attach(handle, m.cardName), "snd_mixer_attach") {
		return
	}
	defer alsa.snd_mixer_detach(handle, m.cardName)
	if err(alsa.snd_mixer_load(handle), "snd_mixer_load") {
		return
	}
	defer alsa.snd_mixer_free(handle)
	if err(alsa.snd_mixer_selem_register(handle, nil, nil), "snd_mixer_selem_register") {
		return
	}
	elem := alsa.snd_mixer_find_selem(handle, sid)
	if elem == nil {
		s.Error(fmt.Errorf("snd_mixer_find_selem NULL"))
		return
	}
	var min, max, vol int64
	var mute int32
	alsa.snd_mixer_selem_get_playback_volume_range(elem, &min, &max)
	for {
		alsa.snd_mixer_selem_get_playback_volume(elem, C.SND_MIXER_SCHN_MONO, &vol)
		alsa.snd_mixer_selem_get_playback_switch(elem, C.SND_MIXER_SCHN_MONO, &mute)
		s.Set(Volume{
			Min:        min,
			Max:        max,
			Vol:        vol,
			Mute:       (mute == 0),
			controller: alsaController{elem},
		})
		errCode := alsa.snd_mixer_wait(handle, -1)
		// 4 == Interrupted system call, try again.
		for errCode == -4 {
			errCode = alsa.snd_mixer_wait(handle, -1)
		}
		if err(errCode, "snd_mixer_wait") {
			return
		}
		if err(alsa.snd_mixer_handle_events(handle), "snd_mixer_handle_events") {
			return
		}
	}
}
