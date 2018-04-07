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

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestCollapsingEmpty(t *testing.T) {
	group := Collapsing()
	group.Collapse()
	assert.True(t, group.Collapsed(), "state check for empty group")
	group.Expand()
	assert.False(t, group.Collapsed(), "state check for empty group")
	group.Toggle()
	assert.True(t, group.Collapsed(), "state check for empty group")

	module := testModule.New(t)
	tester := testModule.NewOutputTester(t, group.Add(module))
	module.Output(bar.NewSegment("test"))
	tester.AssertNoOutput("adding to collapsed group")
}

func TestCollapsingWithModule(t *testing.T) {
	group := Collapsing()
	assert.False(t, group.Collapsed(), "group expanded at start")

	module := testModule.New(t)
	wrapped := group.Add(module)
	module.AssertNotStarted("when wrapped")

	tester := testModule.NewOutputTester(t, wrapped)
	module.AssertStarted("when wrapping module is started")

	out := bar.NewSegment("hello")
	module.Output(out)
	wOut := tester.AssertOutput("passes thru when expanded")
	assert.Equal(t, out.Segments(), wOut, "output is unchanged")

	group.Collapse()
	tester.AssertEmpty("on collapse")

	group.Expand()
	wOut = tester.AssertOutput("on expand")
	assert.Equal(t, out.Segments(), wOut, "original output re-sent")

	group.Toggle()
	assert.True(t, group.Collapsed(), "state check")
	tester.AssertEmpty("on collapse")
	out2 := bar.NewSegment("world")
	module.Output(out2)
	tester.AssertNoOutput("while collapsed")

	group.Toggle()
	assert.False(t, group.Collapsed(), "state check")
	wOut = tester.AssertOutput("on expand")
	assert.Equal(t, out2.Segments(), wOut,
		"output while collapsed is not discarded")

	out3 := bar.NewSegment("foo")
	module.Output(out3)
	wOut = tester.AssertOutput("passes thru when expanded")
	assert.Equal(t, out3.Segments(), wOut, "works normally when expanded")

	wrapped.(bar.Pausable).Pause()
	module.AssertPaused("when wrapper is paused")
	wrapped.(bar.Pausable).Resume()
	module.AssertResumed("when wrapper is resumed")

	evt := bar.Event{X: 1, Y: 1}
	wrapped.(bar.Clickable).Click(evt)
	recvEvt := module.AssertClicked("when wrapper is clicked")
	assert.Equal(t, evt, recvEvt, "click event passed through unchanged")
}

func TestCollapsingButton(t *testing.T) {
	group := Collapsing()
	leftClick := bar.Event{Button: bar.ButtonLeft}
	col := bar.NewSegment("collapsed")
	exp := bar.NewSegment("expanded")

	button := group.Button(col, exp)
	buttonTester := testModule.NewOutputTester(t, button)

	out := buttonTester.AssertOutput("initial output")
	assert.Equal(t, exp.Segments(), out, "starts expanded")

	button.Click(leftClick)
	out = buttonTester.AssertOutput("when clicked")
	assert.Equal(t, col.Segments(), out, "collapsed")
	assert.True(t, group.Collapsed(), "collapsed")

	button.Click(leftClick)
	out = buttonTester.AssertOutput("when clicked")
	assert.Equal(t, exp.Segments(), out, "expanded")
	assert.False(t, group.Collapsed(), "expanded")

	buttonTester.AssertNoOutput("no output without interaction")
}
