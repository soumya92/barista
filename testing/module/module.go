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
	"testing"
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
	assert  *assert.Assertions
	started bool
	outputs chan bar.Output
	pauses  chan bool
	events  chan bar.Event
}

// New creates a new module with the given testingT that can be used
// to assert the behaviour of the bar (or related modules).
func New(t assert.TestingT) *TestModule {
	m := &TestModule{assert: assert.New(t)}
	m.Reset()
	return m
}

// Stream conforms to bar.Module.
func (t *TestModule) Stream() <-chan bar.Output {
	if t.started {
		panic("already streaming!")
	}
	t.started = true
	return t.outputs
}

// Click conforms to bar.Clickable.
func (t *TestModule) Click(e bar.Event) {
	t.events <- e
}

// Pause conforms to bar.Pausable.
func (t *TestModule) Pause() {
	t.pauses <- true
}

// Resume conforms to bar.Pausable.
func (t *TestModule) Resume() {
	t.pauses <- false
}

// Output queues output to be sent over the channel on the next read.
func (t *TestModule) Output(out bar.Output) {
	t.outputs <- out
}

// AssertStarted asserts that the module was started.
func (t *TestModule) AssertStarted(message string) {
	t.assert.True(t.started, message)
}

// AssertNotStarted asserts that the module was not started.
func (t *TestModule) AssertNotStarted(message string) {
	t.assert.False(t.started, message)
}

// AssertPaused asserts that the module was paused,
// and consumes the pause invocation.
func (t *TestModule) AssertPaused(message string) {
	select {
	case state := <-t.pauses:
		t.assert.True(state, message)
	case <-time.After(positiveTimeout):
		t.assert.Fail("expected pause", message)
	}
}

// AssertResumed asserts that the module was resumed,
// and consumes the resume invocation.
func (t *TestModule) AssertResumed(message string) {
	select {
	case state := <-t.pauses:
		t.assert.False(state, message)
	case <-time.After(positiveTimeout):
		t.assert.Fail("expected resume", message)
	}
}

// AssertNoPauseResume asserts that this module had no pause/resume interactions.
func (t *TestModule) AssertNoPauseResume(message string) {
	select {
	case <-t.pauses:
		t.assert.Fail("expected no pause/resume", message)
	case <-time.After(negativeTimeout):
	}
}

// AssertClicked asserts that the module was clicked and returns the event.
// Calling this multiple times asserts multiple click events.
func (t *TestModule) AssertClicked(message string) bar.Event {
	select {
	case evt := <-t.events:
		return evt
	case <-time.After(positiveTimeout):
		t.assert.Fail("expected a click event", message)
		return bar.Event{}
	}
}

// AssertNotClicked asserts that the module received no events.
func (t *TestModule) AssertNotClicked(message string) {
	select {
	case <-t.events:
		t.assert.Fail("expected no click event", message)
	case <-time.After(negativeTimeout):
	}
}

// Reset clears the history of pause/resume/click/stream invocations,
// flushes any buffered events and resets the output channel.
func (t *TestModule) Reset() {
	if t.outputs != nil {
		close(t.outputs)
		close(t.events)
		close(t.pauses)
	}
	t.outputs = make(chan bar.Output, 100)
	t.events = make(chan bar.Event, 100)
	t.pauses = make(chan bool, 100)
	t.started = false
}

// OutputTester groups an output channel and testing.T to simplify
// testing of a bar module.
type OutputTester struct {
	*testing.T
	outs <-chan bar.Output
}

// NewOutputTester creates a started outputTester from the given Module and testing.T.
func NewOutputTester(t *testing.T, m bar.Module) *OutputTester {
	return &OutputTester{t, m.Stream()}
}

// AssertNoOutput asserts that no updates occur on the output channel.
func (o *OutputTester) AssertNoOutput(message string) {
	select {
	case <-o.outs:
		assert.Fail(o, "expected no update", message)
	case <-time.After(negativeTimeout):
	}
}

// AssertOutput asserts that the output channel was updated and returns the output.
func (o *OutputTester) AssertOutput(message string) []bar.Segment {
	select {
	case out := <-o.outs:
		return out.Segments()
	case <-time.After(positiveTimeout):
		assert.Fail(o, "expected an update", message)
		return nil
	}
}

// AssertEmpty asserts that the output channel was updated with empty output.
func (o *OutputTester) AssertEmpty(message string) {
	out := o.AssertOutput(message)
	assert.Empty(o, out, message)
}

// AssertError asserts that the output channel was updated with an error,
// and returns the error string.
func (o *OutputTester) AssertError(message string) string {
	out := o.AssertOutput(message)
	if len(out) != 1 {
		assert.Fail(o, "Expected an error output", message)
		return ""
	}
	urgent, ok := out[0]["urgent"]
	if !ok {
		assert.Fail(o, "Expected an error output", message)
		return ""
	}
	assert.True(o, urgent.(bool), message)
	assert.Equal(o, out[0]["short_text"], "Error", message)
	return out[0].Text()
}

// Drain empties the output channel when the exact number of outputs
// doesn't matter, to allow further testing to start with a clean slate.
func (o *OutputTester) Drain() {
	for {
		select {
		case <-o.outs:
		case <-time.After(negativeTimeout):
			return
		}
	}
}
