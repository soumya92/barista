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
	group := Cycling()

	module1 := testModule.New(t)
	out1 := bar.Output{bar.NewSegment("1")}
	wrapped1 := group.Add(module1)
	module1.AssertNotStarted("when wrapped")

	tester1 := testModule.NewOutputTester(t, wrapped1)
	module1.AssertStarted("when wrapping module is started")
	module1.Output(out1)
	tester1.AssertOutput("first module starts visible")

	module2 := testModule.New(t)
	out2 := bar.Output{bar.NewSegment("2")}
	tester2 := testModule.NewOutputTester(t, group.Add(module2))
	module2.Output(out2)
	tester2.AssertNoOutput("other modules start hidden")

	module3 := testModule.New(t)
	out3 := bar.Output{bar.NewSegment("3")}
	tester3 := testModule.NewOutputTester(t, group.Add(module3))
	module3.Output(out3)
	tester3.AssertNoOutput("other modules start hidden")

	group.Next()
	assert.Equal(t, 1, group.Visible(), "updates visible index")
	tester1.AssertEmpty("on switch")
	wOut := tester2.AssertOutput("shows next module on switch")
	assert.Equal(t, out2, wOut, "updates with previous output")
	tester3.AssertNoOutput("only two modules updated at a time")

	group.Previous()
	assert.Equal(t, 0, group.Visible(), "updates visible index")
	tester2.AssertEmpty("previous output on switch")
	wOut = tester1.AssertOutput("shows next module on switch")
	assert.Equal(t, out1, wOut, "updates with previous output")
	tester3.AssertNoOutput("only two modules updated at a time")

	group.Show(2)
	assert.Equal(t, 2, group.Visible(), "updates visible index")
	tester1.AssertEmpty("previous output on switch")
	wOut = tester3.AssertOutput("shows next module on switch")
	assert.Equal(t, out3, wOut, "updates with previous output")
	tester2.AssertNoOutput("only two modules updated at a time")

	out4 := bar.Output{bar.NewSegment("4")}
	module2.Output(out4)
	tester2.AssertNoOutput("while hidden")
	out5 := bar.Output{bar.NewSegment("5")}
	module2.Output(out5)
	tester2.AssertNoOutput("while hidden")
	group.Show(1)
	wOut = tester2.AssertOutput("when visible")
	assert.Equal(t, out5, wOut, "updates while hidden coalesced")
}

func TestCyclingButton(t *testing.T) {
	group := Cycling()
	leftClick := bar.Event{Button: bar.ButtonLeft}
	scrollUp := bar.Event{Button: bar.ScrollUp}
	for i := 0; i <= 3; i++ {
		group.Add(testModule.New(t))
	}
	button := group.Button(bar.Output{})
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
