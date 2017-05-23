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

// TestModule represents a bar.Module used for testing.
type TestModule struct {
	assert  *assert.Assertions
	outChan chan bar.Output
	sigChan chan interface{}
	started bool
	pauses  []bool
	events  []bar.Event
	outputs []bar.Output
}

// New creates a new module with the given testingT that can be used
// to assert the behaviour of the bar (or related modules).
func New(t assert.TestingT) *TestModule {
	return &TestModule{
		assert:  assert.New(t),
		outChan: make(chan bar.Output),
		sigChan: make(chan interface{}, 1),
	}
}

// Stream conforms to bar.Module.
func (t *TestModule) Stream() <-chan bar.Output {
	if t.started {
		panic("already streaming!")
	}
	go t.looper()
	t.started = true
	return t.outChan
}

// Click conforms to bar.Clickable.
func (t *TestModule) Click(e bar.Event) {
	t.events = append(t.events, e)
}

// Pause conforms to bar.Pausable.
func (t *TestModule) Pause() {
	t.pauses = append(t.pauses, true)
}

// Resume conforms to bar.Pausable.
func (t *TestModule) Resume() {
	t.pauses = append(t.pauses, false)
}

// Output queues output to be sent over the channel on the next read.
func (t *TestModule) Output(out bar.Output) {
	t.outputs = append(t.outputs, out)
	select {
	case t.sigChan <- nil:
	default:
	}
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
	t.assert.True(t.pauses[0], message)
	// clear after assertion so sequential assertions match sequential events.
	t.pauses = t.pauses[1:]
}

// AssertResumed asserts that the module was resumed,
// and consumes the resume invocation.
func (t *TestModule) AssertResumed(message string) {
	t.assert.False(t.pauses[0], message)
	t.pauses = t.pauses[1:]
}

// AssertNoPauseResume asserts that this module had no pause/resume interactions.
func (t *TestModule) AssertNoPauseResume(message string) {
	t.assert.Empty(t.pauses, message)
}

// AssertClicked asserts that the module was clicked with the given event.
// Calling this multiple times asserts multiple click events.
func (t *TestModule) AssertClicked(expected bar.Event, message string) {
	t.assert.Equal(t.events[0], expected, message)
	t.events = t.events[1:]
}

// AssertNotClicked asserts that the module received no events.
func (t *TestModule) AssertNotClicked(message string) {
	t.assert.Empty(t.events, message)
}

// Reset clears the history of pause/resume/click/stream invocations,
// flushes any buffered events and resets the output channel.
func (t *TestModule) Reset() {
	close(t.outChan)
	t.outChan = make(chan bar.Output)
	close(t.sigChan)
	t.sigChan = make(chan interface{})
	t.started = false
	t.pauses = nil
	t.events = nil
	t.outputs = nil
}

func (t *TestModule) looper() {
	for range t.sigChan {
		for len(t.outputs) > 0 {
			t.outChan <- t.outputs[0]
			t.outputs = t.outputs[1:]
		}
	}
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
	case <-time.After(10 * time.Millisecond):
	}
}

// AssertOutput asserts that the output channel was updated and returns the output.
func (o *OutputTester) AssertOutput(message string) bar.Output {
	select {
	case out := <-o.outs:
		return out
	case <-time.After(time.Second):
		assert.Fail(o, "expected an update", message)
		return bar.Output{}
	}
}
