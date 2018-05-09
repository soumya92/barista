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
	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestCollapsingEmpty(t *testing.T) {
	testBar.New(t)

	group := Collapsing()
	group.Collapse()
	assert.True(t, group.Collapsed(), "state check for empty group")
	group.Expand()
	assert.False(t, group.Collapsed(), "state check for empty group")
	group.Toggle()
	assert.True(t, group.Collapsed(), "state check for empty group")

	module := testModule.New(t)
	testBar.Run(group.Add(module))
	module.AssertStarted("when wrapping module is started")
	module.OutputText("test")
	testBar.AssertNoOutput("adding to collapsed group")
}

func TestCollapsingWithModule(t *testing.T) {
	testBar.New(t)

	group := Collapsing()
	assert.False(t, group.Collapsed(), "group expanded at start")

	module := testModule.New(t)
	wrapped := group.Add(module)
	module.AssertNotStarted("when wrapped")

	testBar.Run(wrapped)
	module.AssertStarted("when wrapping module is started")

	out := outputs.Text("hello")
	module.Output(out)
	testBar.NextOutput().AssertEqual(out, "passes thru when expanded")

	group.Collapse()
	testBar.NextOutput().AssertEmpty("on collapse")

	group.Expand()
	testBar.NextOutput().AssertEqual(out, "original output re-sent on expand")

	group.Toggle()
	assert.True(t, group.Collapsed(), "state check")
	testBar.NextOutput().AssertEmpty("on collapse")
	out2 := outputs.Text("world")
	module.Output(out2)
	testBar.AssertNoOutput("while collapsed")

	group.Toggle()
	assert.False(t, group.Collapsed(), "state check")
	testBar.NextOutput().AssertEqual(out2, "output while collapsed is not discarded")

	out3 := outputs.Text("foo")
	module.Output(out3)
	testBar.NextOutput().AssertEqual(out3, "passes thru when expanded")

	evt := bar.Event{X: 1, Y: 1}
	wrapped.(bar.Clickable).Click(evt)
	recvEvt := module.AssertClicked("when wrapper is clicked")
	assert.Equal(t, evt, recvEvt, "click event passed through unchanged")
}

func TestCollapsingButton(t *testing.T) {
	testBar.New(t)

	group := Collapsing()
	leftClick := bar.Event{Button: bar.ButtonLeft}
	col := outputs.Text("collapsed")
	exp := outputs.Text("expanded")

	button := group.Button(col, exp)
	testBar.Run(button)

	testBar.NextOutput().AssertEqual(exp, "initial output")

	button.Click(leftClick)
	testBar.NextOutput().AssertEqual(col, "collapsed when clicked")
	assert.True(t, group.Collapsed(), "collapsed")

	button.Click(leftClick)
	testBar.NextOutput().AssertEqual(exp, "expanded when clicked")
	assert.False(t, group.Collapsed(), "expanded")

	testBar.AssertNoOutput("no output without interaction")
}
