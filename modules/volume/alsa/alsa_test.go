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

package alsa

import (
	"testing"
	"time"
	"unsafe"
	"reflect"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/modules/volume"
	testBar "barista.run/testing/bar"
	"barista.run/testing/notifier"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func singleErrorTest(t *testing.T, setupFn func(*alsaTester)) {
	var value value.ErrorValue
	valSub, done := value.Subscribe()
	defer done()

	alsaT := alsaTest()
	mod := new(alsaModule)
	setupFn(alsaT)

	doneChan := make(chan struct{}, 1)
	go func() {
		mod.Worker(&value)
		doneChan <- struct{}{}
	}()

	notifier.AssertNotified(t, valSub)
	_, err := value.Get()
	require.Error(t, err, "On error")

	notifier.AssertNotified(t, doneChan, "worker exits on error")
}

func TestErrors(t *testing.T) {
	singleErrorTest(t, func(alsaT *alsaTester) {
		alsaT.on_snd_mixer_selem_id_malloc(func(**ctyp_snd_mixer_selem_id_t) int32 {
			return -1
		})
	})
	singleErrorTest(t, func(alsaT *alsaTester) {
		alsaT.on_snd_mixer_open(func(**ctyp_snd_mixer_t, int32) int32 {
			return -1
		})
	})
	singleErrorTest(t, func(alsaT *alsaTester) {
		alsaT.on_snd_mixer_attach(func(*ctyp_snd_mixer_t, string) int32 {
			return -1
		})
	})
	singleErrorTest(t, func(alsaT *alsaTester) {
		alsaT.on_snd_mixer_load(func(*ctyp_snd_mixer_t) int32 {
			return -1
		})
	})
	singleErrorTest(t, func(alsaT *alsaTester) {
		alsaT.on_snd_mixer_selem_register(func(*ctyp_snd_mixer_t, *ctyp_struct_snd_mixer_selem_regopt, **ctyp_snd_mixer_class_t) int32 {
			return -1
		})
	})
	singleErrorTest(t, func(alsaT *alsaTester) {
		alsaT.on_snd_mixer_find_selem(func(*ctyp_snd_mixer_t, *ctyp_snd_mixer_selem_id_t) *ctyp_snd_mixer_elem_t {
			return nil
		})
	})
}

func TestWaitErrors(t *testing.T) {
	var value value.ErrorValue
	valSub, done := value.Subscribe()
	defer done()

	alsaT := alsaTest()
	mod := new(alsaModule)
	alsaT.on_snd_mixer_find_selem(func(*ctyp_snd_mixer_t, *ctyp_snd_mixer_selem_id_t) *ctyp_snd_mixer_elem_t {
		foo := struct{}{}
		return (*ctyp_snd_mixer_elem_t)(unsafe.Pointer(&foo))
	})
	alsaT.on_snd_mixer_wait(func(*ctyp_snd_mixer_t, int32) int32 {
		return -1
	})
	doneChan := make(chan struct{}, 1)
	alsaT.on_snd_mixer_close(func(*ctyp_snd_mixer_t) int32 {
		doneChan <- struct{}{}
		return 0
	})
	go mod.Worker(&value)
	// first volume is successful, ignore.
	notifier.AssertNotified(t, valSub)
	// Only care about second volume (after wait).
	notifier.AssertNotified(t, valSub)
	_, err := value.Get()
	require.Error(t, err, "On error")
	notifier.AssertNoUpdate(t, valSub, "after error")
	notifier.AssertNotified(t, doneChan, "mixer closed on error")

	alsaT = alsaTest()
	mod = new(alsaModule)
	alsaT.on_snd_mixer_find_selem(func(*ctyp_snd_mixer_t, *ctyp_snd_mixer_selem_id_t) *ctyp_snd_mixer_elem_t {
		foo := struct{}{}
		return (*ctyp_snd_mixer_elem_t)(unsafe.Pointer(&foo))
	})
	alsaT.on_snd_mixer_handle_events(func(*ctyp_snd_mixer_t) int32 {
		return -1
	})
	doneChan = make(chan struct{}, 1)
	alsaT.on_snd_mixer_close(func(*ctyp_snd_mixer_t) int32 {
		doneChan <- struct{}{}
		return 0
	})
	go mod.Worker(&value)
	// first volume is successful, ignore.
	notifier.AssertNotified(t, valSub)
	// Only care about second volume (after wait).
	notifier.AssertNotified(t, valSub)
	_, err = value.Get()
	require.Error(t, err, "On error")
	notifier.AssertNoUpdate(t, valSub, "after error")
	notifier.AssertNotified(t, doneChan, "mixer closed on error")
}

type selem struct {
	name  string
	index uint32
}

type elem struct {
	min, max, vol int64
	enabled       int32
}

type handle struct {
	name string
}

func TestAlsaModule(t *testing.T) {
	testBar.New(t)
	alsaT := alsaTest()

	oldRateLimiter := volume.RateLimiter
	defer func() { volume.RateLimiter = oldRateLimiter }()
	volume.RateLimiter = rate.NewLimiter(rate.Inf, 0)

	testHandles := map[*ctyp_snd_mixer_t]chan struct{}{}
	testElems := map[string]*elem{
		"Master": &elem{0, 10, 1, 0},
		"Other":  &elem{0, 65535, 65533, 0},
	}

	alsaT.on_snd_mixer_open(func(ptrToPtr **ctyp_snd_mixer_t, _ int32) int32 {
		handle := new(handle)
		*ptrToPtr = (*ctyp_snd_mixer_t)(unsafe.Pointer(reflect.ValueOf(handle).Pointer()))
		testHandles[*ptrToPtr] = make(chan struct{}, 1)
		return 0
	})
	alsaT.on_snd_mixer_close(func(handle *ctyp_snd_mixer_t) int32 {
		delete(testHandles, handle)
		return 0
	})
	alsaT.on_snd_mixer_selem_id_malloc(func(ptrToPtr **ctyp_snd_mixer_selem_id_t) int32 {
		selem := new(selem)
		*ptrToPtr = (*ctyp_snd_mixer_selem_id_t)(unsafe.Pointer(reflect.ValueOf(selem).Pointer()))
		return 0
	})
	alsaT.on_snd_mixer_selem_id_set_index(func(cptrSelem *ctyp_snd_mixer_selem_id_t, i uint32) {
		ptrSelem := (*selem)(unsafe.Pointer(cptrSelem))
		ptrSelem.index = i
	})
	alsaT.on_snd_mixer_selem_id_set_name(func(cptrSelem *ctyp_snd_mixer_selem_id_t, n string) {
		ptrSelem := (*selem)(unsafe.Pointer(cptrSelem))
		ptrSelem.name = n
	})
	alsaT.on_snd_mixer_selem_get_playback_volume_range(func(cptrElem *ctyp_snd_mixer_elem_t, min *int64, max *int64) int32 {
		ptrElem := (*elem)(unsafe.Pointer(cptrElem))
		*min = ptrElem.min
		*max = ptrElem.max
		return 0
	})
	alsaT.on_snd_mixer_selem_get_playback_volume(func(cptrElem *ctyp_snd_mixer_elem_t, _ ctyp_snd_mixer_selem_channel_id_t, vol *int64) int32 {
		ptrElem := (*elem)(unsafe.Pointer(cptrElem))
		*vol = ptrElem.vol
		return 0
	})
	alsaT.on_snd_mixer_selem_get_playback_switch(func(cptrElem *ctyp_snd_mixer_elem_t, _ ctyp_snd_mixer_selem_channel_id_t, enabled *int32) int32 {
		ptrElem := (*elem)(unsafe.Pointer(cptrElem))
		*enabled = ptrElem.enabled
		return 0
	})
	alsaT.on_snd_mixer_find_selem(func(handle *ctyp_snd_mixer_t, cptrSelem *ctyp_snd_mixer_selem_id_t) *ctyp_snd_mixer_elem_t {
		require.NotNil(t, testHandles[handle], "Handle was not previously registered")
		selem := (*selem)(unsafe.Pointer(cptrSelem))
		e, ok := testElems[selem.name]
		if !ok {
			return nil
		}
		return (*ctyp_snd_mixer_elem_t)(unsafe.Pointer(e))
	})
	alsaT.on_snd_mixer_selem_set_playback_switch_all(func(cptrElem *ctyp_snd_mixer_elem_t, enabled int32) int32 {
		ptrElem := (*elem)(unsafe.Pointer(cptrElem))
		ptrElem.enabled = enabled
		return 0
	})
	alsaT.on_snd_mixer_selem_set_playback_volume_all(func(cptrElem *ctyp_snd_mixer_elem_t, vol int64) int32 {
		ptrElem := (*elem)(unsafe.Pointer(cptrElem))
		ptrElem.vol = vol
		return 0
	})

	alsaT.on_snd_mixer_wait(func(handle *ctyp_snd_mixer_t, _ int32) int32 {
		mixerWait, ok := testHandles[handle]
		require.True(t, ok, "Handle was not previously registered")
		select {
		case <-mixerWait:
			return 0
		case <-time.After(10 * time.Millisecond):
			return -4 // interrupted.
		}
	})

	testBar.Run(volume.New(DefaultMixer()), volume.New(Mixer("default", "Other")))
	testBar.LatestOutput().AssertText([]string{"MUT", "MUT"})

	testElems["Other"].enabled = 1
	for _, ch := range testHandles {
		ch <- struct{}{}
	}

	out := testBar.LatestOutput()
	out.AssertText([]string{"MUT", "100%"})

	out.At(0).Click(bar.Event{Button: bar.ButtonLeft})
	out = testBar.LatestOutput(0)
	out.AssertText([]string{"10%", "100%"})

	out.At(1).Click(bar.Event{Button: bar.ButtonLeft})
	out = testBar.LatestOutput(1)
	out.AssertText([]string{"10%", "MUT"})

	out.At(0).Click(bar.Event{Button: bar.ScrollUp})
	out = testBar.LatestOutput(0)
	out.AssertText([]string{"20%", "MUT"})

	out.At(1).Click(bar.Event{Button: bar.ScrollDown})
	out = testBar.LatestOutput(1)
	out.AssertText([]string{"20%", "MUT"})

	out.At(1).Click(bar.Event{Button: bar.ButtonLeft})
	out = testBar.LatestOutput(1)
	out.AssertText([]string{"20%", "99%"})

	testElems["Other"].vol = 64224
	testElems["Master"].vol = 14
	for _, ch := range testHandles {
		ch <- struct{}{}
	}
	testBar.LatestOutput().AssertText([]string{"140%", "98%"})

	alsaT.on_snd_mixer_close(func(handle *ctyp_snd_mixer_t) int32 {
		close(testHandles[handle])
		return 0
	})
	alsaT.on_snd_mixer_wait(func(*ctyp_snd_mixer_t, int32) int32 {
		return -1
	})
	// Make sure all workers have terminated, so that -count > 1 and -race work.
	for _, ch := range testHandles {
		notifier.AssertClosed(t, ch, "mixer closed on error")
	}
}
