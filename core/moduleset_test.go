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

package core

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

func nextUpdate(t *testing.T, ch <-chan int, formatAndArgs ...interface{}) int {
	select {
	case idx := <-ch:
		return idx
	case <-time.After(time.Second):
		assert.Fail(t, "No update from moduleset", formatAndArgs...)
	}
	return -1
}

func assertNoUpdate(t *testing.T, ch <-chan int, formatAndArgs ...interface{}) {
	select {
	case <-ch:
		assert.Fail(t, "Unexpected update from moduleset", formatAndArgs...)
	case <-time.After(10 * time.Millisecond):
		// test passed.
	}
}

func TestModuleSet(t *testing.T) {
	tms := []*testModule.TestModule{
		testModule.New(t),
		testModule.New(t),
		testModule.New(t),
	}
	ms := NewModuleSet([]bar.Module{tms[0], tms[1], tms[2]})
	updateCh := ms.Stream()
	assertNoUpdate(t, updateCh, "on start")
	for _, tm := range tms {
		tm.AssertStarted("on moduleset stream")
	}

	tms[1].OutputText("foo")
	assert.Equal(t, 1, nextUpdate(t, updateCh, "on output"),
		"update notification on new output from module")

	tms[0].OutputText("baz")
	assert.Equal(t, 0, nextUpdate(t, updateCh, "on update"),
		"update notification on new output from module")

	assert.Empty(t, ms.LastOutput(2), "without any output")
	assert.Equal(t, "baz", ms.LastOutput(0)[0].Text())

	out := ms.LastOutputs()
	assert.Equal(t,
		[]bar.Segments{
			{bar.TextSegment("baz")},
			{bar.TextSegment("foo")},
			nil,
		}, out)

	ms.Click(0, bar.Event{X: 40})
	e := tms[0].AssertClicked("on moduleset click")
	assert.Equal(t, 40, e.X)

	tms[2].Output(outputs.Errorf("something went wrong"))
	<-updateCh

	tms[2].Close()
	assertNoUpdate(t, updateCh, "on module finish")

	ms.Click(2, bar.Event{Button: bar.ScrollUp})
	tms[2].AssertNotClicked("after finish")
	tms[2].AssertNotStarted("when not left/right/middle clicked")

	ms.Click(2, bar.Event{Button: bar.ButtonLeft})
	assert.Equal(t, 2, nextUpdate(t, updateCh, "on restart"),
		"update notification on restart")
	assert.Empty(t, ms.LastOutput(2), "error segment removed")
	tms[2].AssertStarted("on left click after finish")
}
