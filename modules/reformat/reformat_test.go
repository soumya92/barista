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
	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestReformat(t *testing.T) {
	testBar.New(t)
	original := testModule.New(t)
	reformatted := New(original, func(o bar.Output) bar.Output {
		return outputs.Textf("+%s+", o.Segments()[0].Text())
	})
	original.AssertNotStarted("on construction of reformatted module")
	testBar.Run(reformatted)
	original.AssertStarted("on stream of reformatted module")

	original.Output(outputs.Textf("test"))
	testBar.NextOutput().AssertText([]string{"+test+"},
		"when original module updates")

	evt := bar.Event{Y: 1}
	reformatted.(bar.Clickable).Click(evt)
	recvEvt := original.AssertClicked("click events propagated")
	assert.Equal(t, evt, recvEvt, "click events passed through unchanged")

	testBar.AssertNoOutput("when original module is not updated")
	original.AssertNotClicked("when reformatted module is not clicked")
}

func TestRestart(t *testing.T) {
	testBar.New(t)
	original := testModule.New(t)
	reformatted := New(original, func(o bar.Output) bar.Output {
		return outputs.Textf("+%s+", o.Segments()[0].Text())
	})
	testBar.Run(reformatted)
	original.AssertStarted("on stream of reformatted module")

	original.Output(outputs.Textf("test"))
	testBar.NextOutput().AssertText([]string{"+test+"})

	original.Close()
	testBar.AssertNoOutput("on close")
	original.AssertNotStarted("after close")

	testBar.Click(0)
	original.AssertStarted("when reformatted module is clicked")
	testBar.AssertNoOutput("until original module outputs")

	original.OutputText("foo")
	testBar.NextOutput().AssertText([]string{"+foo+"})
}
