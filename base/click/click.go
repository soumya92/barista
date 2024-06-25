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

// Package click provides methods to compose click handlers.
package click

import (
	"os/exec"

	"github.com/soumya92/barista/bar"
)

// DiscardEvent wraps a function with no arguments in a function that takes a
// bar.Event, allowing a func() to be used as a click handler.
func DiscardEvent(fn func()) func(bar.Event) {
	return func(bar.Event) { fn() }
}

// Click invokes the given function on button clicks only (ignoring scroll).
// includeBackAndForward controls whether the back and forward buttons also
// trigger the handler. (Only the first value is used, but varargs provide
// an "optional" argument here.). Defaults to false.
func Click(do func(), includeBackAndForward ...bool) func(bar.Event) {
	btns := []bar.Button{bar.ButtonLeft, bar.ButtonMiddle, bar.ButtonRight}
	if len(includeBackAndForward) > 0 && includeBackAndForward[0] {
		btns = append(btns, bar.ButtonBack, bar.ButtonForward)
	}
	return Button(func(bar.Button) { do() }, btns...)
}

// Scroll invokes the given function on all scroll events, and passes in the
// button (e.g. bar.ScrollUp).
func Scroll(do func(bar.Button)) func(bar.Event) {
	return Button(do,
		bar.ScrollUp, bar.ScrollDown, bar.ScrollLeft, bar.ScrollRight)
}

// Button invokes the given function when any of the specified buttons trigger
// the event handler. It passes only the button to the function. To get the
// complete event, see ButtonE.
func Button(do func(bar.Button), btns ...bar.Button) func(bar.Event) {
	return ButtonE(func(e bar.Event) { do(e.Button) }, btns...)
}

// ButtonE filters out events triggered by buttons not listed in btns before
// invoking the given click handler.
func ButtonE(handler func(bar.Event), btns ...bar.Button) func(bar.Event) {
	btnMap := map[bar.Button]bool{}
	for _, b := range btns {
		btnMap[b] = true
	}
	return func(e bar.Event) {
		if btnMap[e.Button] {
			handler(e)
		}
	}
}

// RunLeft executes the given command on a left-click. This is a shortcut for
// click.Left(func(){exec.Command(cmd).Run()}).
func RunLeft(cmd string, args ...string) func(bar.Event) {
	return Left(func() {
		exec.Command(cmd, args...).Run()
	})
}

// fallbackButton is used as a placeholder for all other buttons.
const fallbackButton = bar.Button(-1)

// Map stores a mapping of button to event handler.
type Map map[bar.Button]func(bar.Event)

// Handle handles an event and invokes the appropriate handler from the map.
func (m Map) Handle(e bar.Event) {
	if handler, ok := m[e.Button]; ok {
		handler(e)
	} else if fallback, ok := m[fallbackButton]; ok {
		fallback(e)
	}
}

// Set sets the click handler for a button, and returns the map for chaining.
func (m Map) Set(btn bar.Button, handler func(bar.Event)) Map {
	m[btn] = handler
	return m
}

// Else sets the click handler for all buttons that don't already have one.
func (m Map) Else(handler func(bar.Event)) Map {
	return m.Set(fallbackButton, handler)
}

// Generate methods for each button, both at the package level and on Map.
//go:generate ruby buttons.rb
