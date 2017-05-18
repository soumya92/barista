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
	wrapped := tester(group.Add(module), t)
	module.Output(bar.Output{bar.NewSegment("test")})
	wrapped.assertNoOutput("adding to collapsed group")
}

func TestCollapsingWithModule(t *testing.T) {
	group := Collapsing()
	assert.False(t, group.Collapsed(), "group expanded at start")

	module := testModule.New(t)
	wrappedModule := group.Add(module)
	module.AssertNotStarted("when wrapped")
	// tester starts the module to obtain the output channel.
	wrapped := tester(wrappedModule, t)
	module.AssertStarted("when wrapping module is started")

	out := bar.Output{bar.NewSegment("hello")}
	module.Output(out)
	wOut := wrapped.assertOutput("passes thru when expanded")
	assert.Equal(t, out, wOut, "output is unchanged")

	group.Collapse()
	wrapped.assertEmpty("clears on collapse")

	group.Expand()
	wOut = wrapped.assertOutput("on expand")
	assert.Equal(t, out, wOut, "original output re-sent")

	group.Toggle()
	assert.True(t, group.Collapsed(), "state check")
	wrapped.assertEmpty("clear on collapse")
	out2 := bar.Output{bar.NewSegment("world")}
	module.Output(out2)
	wrapped.assertNoOutput("while collapsed")

	group.Toggle()
	assert.False(t, group.Collapsed(), "state check")
	wOut = wrapped.assertOutput("on expand")
	assert.Equal(t, out2, wOut, "output while collapsed is not discarded")

	out3 := bar.Output{bar.NewSegment("foo")}
	module.Output(out3)
	wOut = wrapped.assertOutput("passes thru when expanded")
	assert.Equal(t, out3, wOut, "works normally when expanded")

	wrappedModule.(bar.Pausable).Pause()
	module.AssertPaused("when wrapper is paused")
	wrappedModule.(bar.Pausable).Resume()
	module.AssertResumed("when wrapper is resumed")

	evt := bar.Event{X: 1, Y: 1}
	wrappedModule.(bar.Clickable).Click(evt)
	module.AssertClicked(evt, "when wrapper is clicked")
}

func TestCollapsingButton(t *testing.T) {
	group := Collapsing()
	leftClick := bar.Event{Button: bar.ButtonLeft}
	col := bar.Output{bar.NewSegment("col")}
	exp := bar.Output{bar.NewSegment("exp")}

	button := group.Button(col, exp)
	buttonTester := tester(button, t)

	out := buttonTester.assertOutput("initial output")
	assert.Equal(t, exp, out, "starts expanded")

	button.Click(leftClick)
	out = buttonTester.assertOutput("when clicked")
	assert.Equal(t, col, out, "collapsed")
	assert.True(t, group.Collapsed(), "collapsed")

	button.Click(leftClick)
	out = buttonTester.assertOutput("when clicked")
	assert.Equal(t, exp, out, "expanded")
	assert.False(t, group.Collapsed(), "expanded")

	buttonTester.assertNoOutput("no output without interaction")
}
