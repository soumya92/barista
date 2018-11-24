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
  #include <alsa/asoundlib.h>
  #include <alsa/mixer.h>
  #include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"unsafe"

	"barista.run/base/value"
	l "barista.run/logging"
)

// ALSA implementation.
type alsaModule struct {
	cardName  string
	mixerName string
}

type alsaController struct {
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

func (c alsaController) setVolume(newVol int64) error {
	return alsaError(
		C.snd_mixer_selem_set_playback_volume_all(c.elem, C.long(newVol)),
		"snd_mixer_selem_set_playback_volume_all")
}

func (c alsaController) setMuted(muted bool) error {
	var muteInt C.int
	if muted {
		muteInt = C.int(0)
	} else {
		muteInt = C.int(1)
	}
	return alsaError(
		C.snd_mixer_selem_set_playback_switch_all(c.elem, muteInt),
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
	elem := C.snd_mixer_find_selem(handle, sid)
	if elem == nil {
		s.Error(fmt.Errorf("snd_mixer_find_selem NULL"))
		return
	}
	var min, max, vol C.long
	var mute C.int
	C.snd_mixer_selem_get_playback_volume_range(elem, &min, &max)
	for {
		C.snd_mixer_selem_get_playback_volume(elem, C.SND_MIXER_SCHN_MONO, &vol)
		C.snd_mixer_selem_get_playback_switch(elem, C.SND_MIXER_SCHN_MONO, &mute)
		s.Set(Volume{
			Min:        int64(min),
			Max:        int64(max),
			Vol:        int64(vol),
			Mute:       (int(mute) == 0),
			controller: alsaController{elem},
		})
		errCode := C.snd_mixer_wait(handle, -1)
		// 4 == Interrupted system call, try again.
		for int(errCode) == -4 {
			errCode = C.snd_mixer_wait(handle, -1)
		}
		if err(errCode, "snd_mixer_wait") {
			return
		}
		if err(C.snd_mixer_handle_events(handle), "snd_mixer_handle_events") {
			return
		}
	}
}
