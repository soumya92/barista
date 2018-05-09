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

// Package module provides a test module that can be used in tests.
package module

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
)

// Time to wait for events that are expected. Overridden in tests.
var positiveTimeout = time.Second

// Time to wait for events that are not expected.
var negativeTimeout = 10 * time.Millisecond

// TestModule represents a bar.Module used for testing.
type TestModule struct {
	sync.Mutex
	assert  *assert.Assertions
	state   atomic.Value // of testModuleState
	onStart chan<- bool
}

type testModuleState struct {
	started bool
	outputs chan bar.Output
	events  chan bar.Event
}

// New creates a new module with the given testingT that can be used
// to assert the behaviour of the bar (or related modules).
func New(t assert.TestingT) *TestModule {
	return &TestModule{assert: assert.New(t)}
}

func (t *TestModule) getState() testModuleState {
	s, _ := t.state.Load().(testModuleState)
	return s // if conversion failed, zero value is fine.
}

// Stream conforms to bar.Module.
func (t *TestModule) Stream() <-chan bar.Output {
	s := t.getState()
	if s.started {
		panic("already streaming!")
	}

	t.Lock()
	newS := testModuleState{
		outputs: make(chan bar.Output, 100),
		events:  make(chan bar.Event, 100),
		started: true,
	}
	onStart := t.onStart
	t.onStart = nil
	t.state.Store(newS)
	t.Unlock()

	if onStart != nil {
		defer func() { onStart <- true }()
	}
	return newS.outputs
}

// Click conforms to bar.Clickable.
func (t *TestModule) Click(e bar.Event) {
	s := t.getState()
	if !s.started {
		panic("not streaming!")
	}
	s.events <- e
}

// Output queues output to be sent over the channel on the next read.
func (t *TestModule) Output(out bar.Output) {
	s := t.getState()
	if !s.started {
		panic("not streaming!")
	}
	s.outputs <- out
}

// OutputText is shorthand for Output(bar.TextSegment(...)).
func (t *TestModule) OutputText(text string) {
	t.Output(bar.TextSegment(text))
}

// Close closes the module's channels, allowing the bar to restart
// the module on click.
func (t *TestModule) Close() {
	s := t.getState()
	close(s.outputs)
	close(s.events)
	t.state.Store(testModuleState{})
}

// AssertStarted waits for the module to start, or does nothing
// if the module is already streaming.
func (t *TestModule) AssertStarted(args ...interface{}) {
	t.Lock()
	if t.getState().started {
		t.Unlock()
		return
	}
	ch := make(chan bool)
	t.onStart = ch
	t.Unlock()

	select {
	case <-ch:
	case <-time.After(positiveTimeout):
		t.assert.Fail("module did not start", args...)
	}
}

// AssertNotStarted asserts that the module was not started.
func (t *TestModule) AssertNotStarted(args ...interface{}) {
	s := t.getState()
	t.assert.False(s.started, args...)
}

// AssertClicked asserts that the module was clicked and returns the event.
// Calling this multiple times asserts multiple click events.
func (t *TestModule) AssertClicked(args ...interface{}) bar.Event {
	s := t.getState()
	select {
	case evt := <-s.events:
		return evt
	case <-time.After(positiveTimeout):
		t.assert.Fail("expected a click event", args...)
		return bar.Event{}
	}
}

// AssertNotClicked asserts that the module received no events.
func (t *TestModule) AssertNotClicked(args ...interface{}) {
	s := t.getState()
	select {
	case <-s.events:
		t.assert.Fail("expected no click event", args...)
	case <-time.After(negativeTimeout):
	}
}
