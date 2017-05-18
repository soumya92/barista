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
	wrappedModule1 := group.Add(module1)
	module1.AssertNotStarted("when wrapped")
	// tester starts the module to obtain the output channel.
	wrapped1 := tester(wrappedModule1, t)
	module1.AssertStarted("when wrapping module is started")
	module1.Output(out1)
	wrapped1.assertOutput("first module starts visible")

	module2 := testModule.New(t)
	out2 := bar.Output{bar.NewSegment("2")}
	wrapped2 := tester(group.Add(module2), t)
	module2.Output(out2)
	wrapped2.assertNoOutput("other modules start hidden")

	module3 := testModule.New(t)
	out3 := bar.Output{bar.NewSegment("3")}
	wrapped3 := tester(group.Add(module3), t)
	module3.Output(out3)
	wrapped3.assertNoOutput("other modules start hidden")

	group.Next()
	assert.Equal(t, 1, group.Visible(), "updates visible index")
	wrapped1.assertEmpty("clears previous module on switch")
	wOut := wrapped2.assertOutput("shows next module on switch")
	assert.Equal(t, out2, wOut, "updates with previous output")
	wrapped3.assertNoOutput("only two modules updated at a time")

	group.Previous()
	assert.Equal(t, 0, group.Visible(), "updates visible index")
	wrapped2.assertEmpty("clears previous module on switch")
	wOut = wrapped1.assertOutput("shows next module on switch")
	assert.Equal(t, out1, wOut, "updates with previous output")
	wrapped3.assertNoOutput("only two modules updated at a time")

	group.Show(2)
	assert.Equal(t, 2, group.Visible(), "updates visible index")
	wrapped1.assertEmpty("clears previous module on switch")
	wOut = wrapped3.assertOutput("shows next module on switch")
	assert.Equal(t, out3, wOut, "updates with previous output")
	wrapped2.assertNoOutput("only two modules updated at a time")

	out4 := bar.Output{bar.NewSegment("4")}
	module2.Output(out4)
	wrapped2.assertNoOutput("while hidden")
	out5 := bar.Output{bar.NewSegment("5")}
	module2.Output(out5)
	wrapped2.assertNoOutput("while hidden")
	group.Show(1)
	wOut = wrapped2.assertOutput("when visible")
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
