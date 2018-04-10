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
	"github.com/soumya92/barista/outputs"
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
	module.Output(outputs.Text("test"))
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

	out := outputs.Text("hello")
	module.Output(out)
	tester.AssertOutputEquals(out, "passes thru when expanded")

	group.Collapse()
	tester.AssertEmpty("on collapse")

	group.Expand()
	tester.AssertOutputEquals(out, "original output re-sent on expand")

	group.Toggle()
	assert.True(t, group.Collapsed(), "state check")
	tester.AssertEmpty("on collapse")
	out2 := outputs.Text("world")
	module.Output(out2)
	tester.AssertNoOutput("while collapsed")

	group.Toggle()
	assert.False(t, group.Collapsed(), "state check")
	tester.AssertOutputEquals(out2, "output while collapsed is not discarded")

	out3 := outputs.Text("foo")
	module.Output(out3)
	tester.AssertOutputEquals(out3, "passes thru when expanded")

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
	col := outputs.Text("collapsed")
	exp := outputs.Text("expanded")

	button := group.Button(col, exp)
	buttonTester := testModule.NewOutputTester(t, button)

	buttonTester.AssertOutputEquals(exp, "initial output")

	button.Click(leftClick)
	buttonTester.AssertOutputEquals(col, "collapsed when clicked")
	assert.True(t, group.Collapsed(), "collapsed")

	button.Click(leftClick)
	buttonTester.AssertOutputEquals(exp, "expanded when clicked")
	assert.False(t, group.Collapsed(), "expanded")

	buttonTester.AssertNoOutput("no output without interaction")
}
