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

package group

import (
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
)

// outputTester groups the output channel and testing.T to simplify
// testing of a module's output.
type outputTester struct {
	*testing.T
	outs <-chan bar.Output
}

// tester creates a started outputTester from the given Module and testing.T.
func tester(m bar.Module, t *testing.T) *outputTester {
	return &outputTester{t, m.Stream()}
}

// assertNoOutput asserts that no updates occur on the output channel.
func (o *outputTester) assertNoOutput(message string) {
	select {
	case <-o.outs:
		assert.Fail(o, "expected no update", message)
	case <-time.After(10 * time.Millisecond):
	}
}

// assertOutput asserts that the output channel was updated and returns the output.
func (o *outputTester) assertOutput(message string) bar.Output {
	select {
	case out := <-o.outs:
		return out
	case <-time.After(time.Second):
		assert.Fail(o, "expected an update", message)
		return bar.Output{}
	}
}

// assertEmpty asserts that the output channel was updated with empty output.
func (o *outputTester) assertEmpty(message string) {
	select {
	case out := <-o.outs:
		assert.Empty(o, out, message)
	case <-time.After(time.Second):
		assert.Fail(o, "expected an update", message)
	}
}

type simpleModule chan bar.Output

func (s simpleModule) Stream() <-chan bar.Output { return (<-chan bar.Output)(s) }

type pausableModule chan bar.Output

func (p pausableModule) Stream() <-chan bar.Output { return (<-chan bar.Output)(p) }
func (p pausableModule) Pause()                    {}
func (p pausableModule) Resume()                   {}

type clickableModule chan bar.Output

func (c clickableModule) Stream() <-chan bar.Output { return (<-chan bar.Output)(c) }
func (c clickableModule) Click(e bar.Event)         {}

func TestWrappedModule(t *testing.T) {
	evt := bar.Event{X: 1, Y: 1}
	for _, m := range []bar.Module{
		make(simpleModule),
		make(pausableModule),
		make(clickableModule),
	} {
		var wrapped WrappedModule = &module{Module: m}
		wrapped.Stream()
		assert.NotPanics(t, wrapped.Pause)
		assert.NotPanics(t, wrapped.Resume)
		assert.NotPanics(t, func() { wrapped.Click(evt) })
	}
}
