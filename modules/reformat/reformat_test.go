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

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestReformat(t *testing.T) {
	testBar.New(t)
	original := testModule.New(t)
	reformatted := New(original)
	original.AssertNotStarted("on construction of reformatted module")
	testBar.Run(reformatted)
	original.AssertStarted("on stream of reformatted module")

	original.Output(outputs.Textf("test"))
	testBar.NextOutput().AssertText([]string{"test"},
		"when original module updates")

	reformatted.Format(func(s bar.Segments) bar.Output {
		return outputs.Textf("+%s+", s[0].Text())
	})
	testBar.NextOutput().AssertText([]string{"+test+"},
		"when format function changes")

	original.Output(outputs.Textf("foo"))
	testBar.NextOutput().AssertText([]string{"+foo+"},
		"when original module updates after format func is set")

	evt := bar.Event{Y: 1}
	reformatted.Click(evt)
	recvEvt := original.AssertClicked("click events propagated")
	require.Equal(t, evt, recvEvt, "click events passed through unchanged")

	testBar.AssertNoOutput("when original module is not updated")
	original.AssertNotClicked("when reformatted module is not clicked")

	reformatted.Format(nil)
	testBar.NextOutput().AssertText([]string{"foo"},
		"nil format passes output through unchanged")

	reformatted.Format(Hide)
	testBar.NextOutput().AssertText([]string{},
		"reformat.Hide hides the output")

	original.Output(outputs.Textf("test"))
	testBar.NextOutput().AssertText([]string{},
		"when original module updates with reformat.Hide")

	reformatted.Format(EachSegment(SkipErrors(
		func(in *bar.Segment) *bar.Segment {
			return outputs.Textf("#%s#", in.Text())
		})))
	testBar.NextOutput().AssertText([]string{"#test#"},
		"with EachSegment wrapper")

	original.Output(outputs.Group(
		outputs.Text("a"), outputs.Text("b"), outputs.Errorf("c")))
	out := testBar.NextOutput()
	out.At(0).AssertText("#a#")
	out.At(1).AssertText("#b#")
	err := out.At(2).AssertError()
	require.Equal(t, "c", err, "erro string unchanged")
}

func TestRestart(t *testing.T) {
	testBar.New(t)
	original := testModule.New(t)
	reformatted := New(original).Format(func(s bar.Segments) bar.Output {
		return outputs.Textf("+%s+", s[0].Text())
	})
	testBar.Run(reformatted)
	original.AssertStarted("on stream of reformatted module")

	original.Output(outputs.Textf("test"))
	testBar.NextOutput().AssertText([]string{"+test+"})

	reformatted.Format(EachSegment(SkipErrors(
		func(s *bar.Segment) *bar.Segment {
			return outputs.Textf("+%s+", s.Text())
		})))
	testBar.NextOutput().Expect("On format func change")

	original.Output(outputs.Group(
		outputs.Errorf("foo"),
		outputs.Text("test"),
	))

	out := testBar.NextOutput()
	out.At(0).AssertError()
	out.At(1).AssertText("+test+")

	original.Close()
	testBar.AssertNoOutput("on close")
	original.AssertNotStarted("after close")

	require.NotPanics(t, func() {
		testBar.SendEvent(0, bar.Event{Button: bar.ScrollUp})
	})

	testBar.Click(0)
	original.AssertStarted("when reformatted module is clicked")
	testBar.NextOutput().AssertText([]string{"+test+"},
		"error segments removed on restart")

	testBar.AssertNoOutput("until original module outputs")
	reformatted.Format(EachSegment(
		func(in *bar.Segment) *bar.Segment {
			return outputs.Textf("** %s **", in.Text())
		}))
	testBar.AssertNoOutput("if format func changes before any output")

	original.OutputText("foo")
	testBar.NextOutput().AssertText([]string{"** foo **"},
		"error segments removed on restart")

	require.NotPanics(t, func() { original.Output(nil) },
		"nil output with EachSegment formatter")
	testBar.NextOutput().AssertEmpty()
}
