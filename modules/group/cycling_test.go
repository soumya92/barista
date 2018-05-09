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

func TestCyclingEmpty(t *testing.T) {
	group := Cycling()
	assert.Equal(t, 0, group.Count(), "empty group has 0 modules")
	group.Show(1)
	assert.Equal(t, 0, group.Visible(), "empty group shows 0th module")
	group.Next()
	assert.Equal(t, 0, group.Visible(), "empty group shows 0th module")
	group.Previous()
	assert.Equal(t, 0, group.Visible(), "empty group shows 0th module")
}

func TestCyclingWithModules(t *testing.T) {
	testBar.New(t)
	group := Cycling()

	module1 := testModule.New(t)
	module2 := testModule.New(t)
	module3 := testModule.New(t)

	wrapped1 := group.Add(module1)
	module1.AssertNotStarted("when wrapped")

	testBar.Run(wrapped1, group.Add(module2), group.Add(module3))
	module1.AssertStarted("when wrapping module is started")
	module2.AssertStarted("when wrapping module is started")
	module3.AssertStarted("when wrapping module is started")
	// implicitly asserts that all modules are started.
	testBar.AssertNoOutput("before any output from module")

	out1 := outputs.Text("1")
	module1.Output(out1)

	out2 := outputs.Text("2")
	module2.Output(out2)

	out3 := outputs.Text("3")
	module3.Output(out3)

	testBar.LatestOutput().AssertEqual(out1, "first module starts visible")

	group.Next()
	assert.Equal(t, 1, group.Visible(), "updates visible index")
	testBar.LatestOutput().AssertEqual(out2, "on switch")

	group.Previous()
	assert.Equal(t, 0, group.Visible(), "updates visible index")
	testBar.LatestOutput().AssertEqual(out1, "previous output on switch")

	group.Show(2)
	assert.Equal(t, 2, group.Visible(), "updates visible index")
	testBar.LatestOutput().AssertEqual(out3, "directly jumping to an index")

	out4 := outputs.Text("4")
	module2.Output(out4)
	testBar.AssertNoOutput("while hidden")

	out5 := outputs.Text("5")
	module2.Output(out5)
	testBar.AssertNoOutput("while hidden")

	group.Show(1)
	testBar.LatestOutput().AssertEqual(out5, "updates while hidden coalesced")
}

func TestCyclingRestart(t *testing.T) {
	testBar.New(t)

	group := Cycling()
	module1 := testModule.New(t)
	module2 := testModule.New(t)
	module3 := testModule.New(t)

	testBar.Run(
		group.Add(module1),
		group.Add(module2),
		group.Add(module3),
	)
	module1.AssertStarted("when wrapping module is started")
	module2.AssertStarted("when wrapping module is started")
	module3.AssertStarted("when wrapping module is started")
	// implicitly asserts that all modules are started.

	module1.OutputText("1")
	module1.Close()

	module2.OutputText("2")
	module2.Close()

	module3.OutputText("3")

	testBar.LatestOutput().AssertText(
		[]string{"1"}, "first module")
	module1.AssertNotStarted("after being stopped")

	group.Next()
	testBar.LatestOutput().AssertText(
		[]string{"2"}, "switching to second module")
	module2.AssertNotStarted("after being stopped")

	// module 2 is showing.
	testBar.Click(0)
	module2.AssertStarted("when clicked after finish")
	assert.NotPanics(t, func() {
		module2.OutputText("2")
	})
	module1.AssertNotStarted("click when not showing")
	testBar.LatestOutput().AssertText([]string{"2"})

	group.Show(2)
	testBar.LatestOutput().Expect("on switch")
	testBar.Click(0)
	// Clicking on the third module does nothing.

	module1.AssertNotStarted("click when not showing")

	group.Show(0)
	testBar.LatestOutput().AssertText([]string{"1"})
	module1.AssertNotStarted("until clicked")

	testBar.Click(0)
	module1.AssertStarted("when clicked after finish")
}

func TestCyclingButton(t *testing.T) {
	testBar.New(t)

	group := Cycling()
	leftClick := bar.Event{Button: bar.ButtonLeft}
	scrollUp := bar.Event{Button: bar.ScrollUp}
	var ts []*testModule.TestModule
	var ms []bar.Module
	for i := 0; i <= 3; i++ {
		tm := testModule.New(t)
		ts = append(ts, tm)
		ms = append(ms, group.Add(tm))
	}
	testBar.Run(ms...)
	for i := 0; i <= 3; i++ {
		ts[i].AssertStarted("when wrapping module is started")
	}
	button := group.Button(outputs.Text("<>"))
	assert.Equal(t, 0, group.Visible(), "starts with first module")
	button.Click(leftClick)
	assert.Equal(t, 1, group.Visible(), "switches to next module on click")
	button.Click(leftClick)
	button.Click(leftClick)
	assert.Equal(t, 3, group.Visible(), "switches to next module on click")
	button.Click(leftClick)
	assert.Equal(t, 0, group.Visible(), "wraps around at the end")
	button.Click(scrollUp)
	assert.Equal(t, 3, group.Visible(), "wraps around to last at beginning")
}
