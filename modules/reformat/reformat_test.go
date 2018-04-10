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

package reformat

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestReformat(t *testing.T) {
	original := testModule.New(t)
	reformatted := New(original, func(o bar.Output) bar.Output {
		return outputs.Textf("+%s+", o.Segments()[0].Text())
	})
	original.AssertNotStarted("on construction of reformatted module")
	tester := testModule.NewOutputTester(t, reformatted)
	original.AssertStarted("on stream of reformatted module")

	original.Output(outputs.Textf("test"))
	tester.AssertOutputEquals(
		outputs.Text("+test+"), "when original module updates")

	reformatted.(bar.Pausable).Pause()
	original.AssertPaused("when reformatted module is paused")

	reformatted.(bar.Pausable).Resume()
	original.AssertResumed("when reformatted module is resumed")

	evt := bar.Event{Y: 1}
	reformatted.(bar.Clickable).Click(evt)
	recvEvt := original.AssertClicked("click events propagated")
	assert.Equal(t, evt, recvEvt, "click events passed through unchanged")

	tester.AssertNoOutput("when original module is not updated")
	original.AssertNoPauseResume("when reformatted module is not paused/resumed")
	original.AssertNotClicked("when reformatted module is not clicked")
}
