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

// Package volume provides an i3bar module that interfaces with alsa or pulse
// to display and control the system volume.
package volume // import "barista.run/modules/volume"

/*
  #cgo pkg-config: alsa
  #include <alsa/asoundlib.h>
  #include <alsa/mixer.h>
  #include <stdlib.h>
*/
import "C"
import (
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	l "barista.run/logging"
	"barista.run/outputs"

	"golang.org/x/time/rate"
)

// Volume represents the current audio volume and mute state.
type Volume struct {
	Min, Max, Vol int64
	Mute          bool
	controller    controller
}

// Frac returns the current volume as a fraction of the total range.
func (v Volume) Frac() float64 {
	return float64(v.Vol-v.Min) / float64(v.Max-v.Min)
}

// Pct returns the current volume in the range 0-100.
func (v Volume) Pct() int {
	return int((v.Frac() * 100) + 0.5)
}

// SetVolume sets the system volume.
// It does not change the mute status.
func (v Volume) SetVolume(volume int64) {
	if volume > v.Max {
		volume = v.Max
	}
	if volume < v.Min {
		volume = v.Min
	}
	if volume == v.Vol {
		return
	}
	if err := v.controller.setVolume(v, volume); err != nil {
		l.Log("Error updating volume: %v", err)
	}
}

// SetMuted controls whether the system volume is muted.
func (v Volume) SetMuted(muted bool) {
	if v.Mute == muted {
		return
	}
	if err := v.controller.setMuted(v, muted); err != nil {
		l.Log("Error updating mute state: %v", err)
	}
}

type controller interface {
	setVolume(Volume, int64) error
	setMuted(Volume, bool) error
}

// Interface that must be implemented by individual volume implementations.
type moduleImpl interface {
	// Infinite loop: push updates and errors to the provided ErrorValue.
	worker(s *value.ErrorValue)
}

// Module represents a bar.Module that displays volume information.
type Module struct {
	outputFunc value.Value // of func(Volume) bar.Output
	impl       moduleImpl
}

// Output configures a module to display the output of a user-defined
// function.
func (m *Module) Output(outputFunc func(Volume) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Throttle volume updates to once every ~20ms to avoid unexpected behaviour.
var rateLimiter = rate.NewLimiter(rate.Every(20*time.Millisecond), 1)

// defaultClickHandler provides a simple example of the click handler capabilities.
// It toggles mute on left click, and raises/lowers the volume on scroll.
func defaultClickHandler(v Volume) func(bar.Event) {
	return func(e bar.Event) {
		if !rateLimiter.Allow() {
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
	var vol value.ErrorValue
	go m.impl.worker(&vol)

	v, err := vol.Get()
	nextV, done := vol.Subscribe()
	defer done()

	outputFunc := m.outputFunc.Get().(func(Volume) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()

	for {
		if s.Error(err) {
			return
		}
		if vol, ok := v.(Volume); ok {
			s.Output(outputs.Group(outputFunc(vol)).
				OnClick(defaultClickHandler(vol)))
		}
		select {
		case <-nextV:
			v, err = vol.Get()
		case <-nextOutputFunc:
			outputFunc = m.outputFunc.Get().(func(Volume) bar.Output)
		}
	}
}

// createModule creates a new module with the given backing implementation.
func createModule(impl moduleImpl) *Module {
	m := &Module{impl: impl}
	l.Register(m, "outputFunc", "impl")
	// Default output is just the volume %, "MUT" when muted.
	m.Output(func(v Volume) bar.Output {
		if v.Mute {
			return outputs.Text("MUT")
		}
		return outputs.Textf("%d%%", v.Pct())
	})
	return m
}
