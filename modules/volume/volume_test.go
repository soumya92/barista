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

import (
	"errors"
	"sync"
	"testing"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"

	"golang.org/x/time/rate"
)

type testVolumeImpl struct {
	sync.Mutex
	error
	min, max, vol int64
	mute          bool
	volChan       chan int64
	muteChan      chan bool
}

func (t *testVolumeImpl) setVolume(vol int64) error {
	t.Lock()
	defer t.Unlock()
	if t.error != nil {
		return t.error
	}
	t.vol = vol
	return nil
}

func (t *testVolumeImpl) setMuted(mute bool) error {
	t.Lock()
	defer t.Unlock()
	if t.error != nil {
		return t.error
	}
	t.mute = mute
	return nil
}

func (t *testVolumeImpl) setError(e error) {
	t.Lock()
	defer t.Unlock()
	t.error = e
}

func (t *testVolumeImpl) worker(v *value.ErrorValue) {
	t.Lock()
	for {
		v.SetOrError(Volume{
			Min:        t.min,
			Max:        t.max,
			Vol:        t.vol,
			Mute:       t.mute,
			controller: t,
		}, t.error)
		t.Unlock()
		select {
		case newVol := <-t.volChan:
			t.Lock()
			t.vol = newVol
		case muted := <-t.muteChan:
			t.Lock()
			t.mute = muted
		}
	}
}

func TestModule(t *testing.T) {
	testBar.New(t)
	testImpl := &testVolumeImpl{
		min: 0, max: 50, vol: 40, mute: false,
		volChan: make(chan int64, 1), muteChan: make(chan bool, 1),
	}
	v := createModule(testImpl)

	testBar.Run(v)

	out := testBar.NextOutput("on start")
	out.AssertText([]string{"80%"})

	out.At(0).LeftClick()
	out = testBar.NextOutput("on click")
	out.AssertText([]string{"MUT"})

	out.At(0).LeftClick()
	testBar.AssertNoOutput("click within 20ms")

	// To speed up the tests.
	rateLimiter = rate.NewLimiter(rate.Inf, 0)

	out.At(0).Click(bar.Event{Button: bar.ScrollUp})
	out = testBar.NextOutput("on volume change")
	out.AssertText([]string{"MUT"}, "still muted")

	out.At(0).Click(bar.Event{Button: bar.ButtonLeft})
	out = testBar.NextOutput("on unmute")
	out.AssertText([]string{"82%"}, "volume value updated")

	testImpl.volChan <- -1
	out = testBar.NextOutput("exernal value update")
	out.AssertText([]string{"-1%"}, "vol < min")

	out.At(0).Click(bar.Event{Button: bar.ScrollDown})
	out = testBar.NextOutput("on volume change")
	out.AssertText([]string{"0%"}, "lower volume at <0%")

	testImpl.volChan <- 100
	out = testBar.NextOutput("exernal value update")
	out.AssertText([]string{"200%"}, "vol > max")

	out.At(0).Click(bar.Event{Button: bar.ScrollDown})
	out = testBar.NextOutput("on volume change")
	out.AssertText([]string{"100%"}, "raise volume at >100%")

	testImpl.setError(errors.New("foo"))

	out.At(0).Click(bar.Event{Button: bar.ScrollDown})
	testBar.AssertNoOutput("error during volume change")

	out.At(0).Click(bar.Event{Button: bar.ButtonLeft})
	testBar.AssertNoOutput("error during mute")

	testImpl.setError(nil)

	v.Output(func(vol Volume) bar.Output {
		return outputs.Textf("%d<%d<%d (%v)", vol.Min, vol.Vol, vol.Max, vol.Mute).
			OnClick(func(e bar.Event) {
				switch e.Button {
				case bar.ButtonLeft:
					vol.SetVolume(0)
				case bar.ButtonMiddle:
					vol.SetVolume(25)
				case bar.ButtonRight:
					vol.SetVolume(50)
				case bar.ScrollDown:
					vol.SetMuted(true)
				case bar.ScrollUp:
					vol.SetMuted(false)
				}
			})
	})

	out = testBar.NextOutput("on output format change")

	out.At(0).Click(bar.Event{Button: bar.ButtonMiddle})
	out = testBar.NextOutput("on volume = 25")

	out.At(0).Click(bar.Event{Button: bar.ButtonMiddle})
	testBar.AssertNoOutput("volume already 25")

	out.At(0).Click(bar.Event{Button: bar.ScrollUp})
	testBar.AssertNoOutput("volume already unmuted")

	out.At(0).Click(bar.Event{Button: bar.ScrollDown})
	out = testBar.NextOutput("on mute")

	out.At(0).Click(bar.Event{Button: bar.ButtonMiddle})
	testBar.AssertNoOutput("volume already 25")

	testImpl.setError(errors.New("some error"))
	testImpl.muteChan <- true

	testBar.NextOutput("on error").AssertError()
}
