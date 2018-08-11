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
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
)

// Time to wait for events that are expected. Overridden in tests.
var positiveTimeout = time.Second

// Time to wait for events that are not expected.
var negativeTimeout = 10 * time.Millisecond

// TestModule represents a bar.Module used for testing.
type TestModule struct {
	sync.Mutex
	require *require.Assertions
	started bool
	outputs chan bar.Output
	events  chan bar.Event
	onStart chan<- bool
	onStop  chan<- bool
}

// New creates a new module with the given testingT that can be used
// to assert the behaviour of the bar (or related modules).
func New(t require.TestingT) *TestModule {
	return &TestModule{require: require.New(t)}
}

// Stream conforms to bar.Module.
func (t *TestModule) Stream(sink bar.Sink) {
	t.Lock()
	if t.started {
		t.Unlock()
		panic("already streaming!")
	}
	outputs := make(chan bar.Output, 100)
	t.outputs = outputs
	t.events = make(chan bar.Event, 100)
	t.started = true
	onStart := t.onStart
	t.onStart = nil
	t.Unlock()

	if onStart != nil {
		go func() { onStart <- true }()
	}
	for out := range outputs {
		sink.Output(out)
	}
	t.Lock()
	t.started = false
	t.Unlock()
}

// Click conforms to bar.Clickable.
func (t *TestModule) Click(e bar.Event) {
	t.Lock()
	defer t.Unlock()
	if !t.started {
		panic(fmt.Sprintf("not streaming! (tried to click: %+#v)", e))
	}
	t.events <- e
}

// Output queues output to be sent over the channel on the next read.
func (t *TestModule) Output(out bar.Output) {
	t.Lock()
	defer t.Unlock()
	if !t.started {
		panic(fmt.Sprintf("not streaming! (tried to output: %+#v)", out))
	}
	t.outputs <- out
}

// OutputText is shorthand for Output(bar.TextSegment(...)).
func (t *TestModule) OutputText(text string) {
	t.Output(bar.TextSegment(text))
}

// ModuleFinished should be called when the module host has processed
// the return from this module's Stream().
func (t *TestModule) ModuleFinished() {
	t.Lock()
	onStop := t.onStop
	t.onStop = nil
	t.Unlock()
	if onStop != nil {
		go func() { onStop <- true }()
	}
}

// Close closes the module's channels, allowing the bar to restart
// the module on click.
func (t *TestModule) Close() {
	stopChan := make(chan bool)
	t.Lock()
	t.onStop = stopChan
	close(t.outputs)
	t.outputs = nil
	t.Unlock()
	<-stopChan
	t.Lock()
	close(t.events)
	t.events = nil
	t.Unlock()
}

// AssertStarted waits for the module to start, or does nothing
// if the module is already streaming.
func (t *TestModule) AssertStarted(args ...interface{}) {
	t.Lock()
	if t.started {
		t.Unlock()
		return
	}
	ch := make(chan bool)
	t.onStart = ch
	t.Unlock()

	select {
	case <-ch:
	case <-time.After(positiveTimeout):
		t.require.Fail("module did not start", args...)
	}
}

// AssertNotStarted asserts that the module was not started.
func (t *TestModule) AssertNotStarted(args ...interface{}) {
	t.Lock()
	defer t.Unlock()
	t.require.False(t.started, args...)
}

// AssertClicked asserts that the module was clicked and returns the event.
// Calling this multiple times asserts multiple click events.
func (t *TestModule) AssertClicked(args ...interface{}) bar.Event {
	t.Lock()
	started := t.started
	evtChan := t.events
	t.Unlock()
	if !started {
		t.require.Fail("expecting click event on stopped module!", args...)
		return bar.Event{}
	}
	select {
	case evt := <-evtChan:
		return evt
	case <-time.After(positiveTimeout):
		t.require.Fail("expected a click event", args...)
		return bar.Event{}
	}
}

// AssertNotClicked asserts that the module received no events.
func (t *TestModule) AssertNotClicked(args ...interface{}) {
	t.Lock()
	evtChan := t.events
	t.Unlock()
	select {
	case <-evtChan:
		t.require.Fail("expected no click event", args...)
	case <-time.After(negativeTimeout):
	}
}
