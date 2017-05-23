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
	module.Output(bar.Output{bar.NewSegment("test")})
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

	out := bar.Output{bar.NewSegment("hello")}
	module.Output(out)
	wOut := tester.AssertOutput("passes thru when expanded")
	assert.Equal(t, out, wOut, "output is unchanged")

	group.Collapse()
	assert.Empty(t, tester.AssertOutput("on collapse"), "clears output")

	group.Expand()
	wOut = tester.AssertOutput("on expand")
	assert.Equal(t, out, wOut, "original output re-sent")

	group.Toggle()
	assert.True(t, group.Collapsed(), "state check")
	out = tester.AssertOutput("on collapse")
	assert.Empty(t, out, "clears on collapse")
	out2 := bar.Output{bar.NewSegment("world")}
	module.Output(out2)
	tester.AssertNoOutput("while collapsed")

	group.Toggle()
	assert.False(t, group.Collapsed(), "state check")
	wOut = tester.AssertOutput("on expand")
	assert.Equal(t, out2, wOut, "output while collapsed is not discarded")

	out3 := bar.Output{bar.NewSegment("foo")}
	module.Output(out3)
	wOut = tester.AssertOutput("passes thru when expanded")
	assert.Equal(t, out3, wOut, "works normally when expanded")

	wrapped.(bar.Pausable).Pause()
	module.AssertPaused("when wrapper is paused")
	wrapped.(bar.Pausable).Resume()
	module.AssertResumed("when wrapper is resumed")

	evt := bar.Event{X: 1, Y: 1}
	wrapped.(bar.Clickable).Click(evt)
	module.AssertClicked(evt, "when wrapper is clicked")
}

func TestCollapsingButton(t *testing.T) {
	group := Collapsing()
	leftClick := bar.Event{Button: bar.ButtonLeft}
	col := bar.Output{bar.NewSegment("col")}
	exp := bar.Output{bar.NewSegment("exp")}

	button := group.Button(col, exp)
	buttonTester := testModule.NewOutputTester(t, button)

	out := buttonTester.AssertOutput("initial output")
	assert.Equal(t, exp, out, "starts expanded")

	button.Click(leftClick)
	out = buttonTester.AssertOutput("when clicked")
	assert.Equal(t, col, out, "collapsed")
	assert.True(t, group.Collapsed(), "collapsed")

	button.Click(leftClick)
	out = buttonTester.AssertOutput("when clicked")
	assert.Equal(t, exp, out, "expanded")
	assert.False(t, group.Collapsed(), "expanded")

	buttonTester.AssertNoOutput("no output without interaction")
}
