// Copyright 2018 Google Inc.
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

package split

import (
	"testing"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	testModule "barista.run/testing/module"
	"github.com/stretchr/testify/require"
)

func TestSplit(t *testing.T) {
	testBar.New(t)

	original := testModule.New(t)
	first, rest := SplitModule(original, 2)

	// To delineate the portions of the output coming from first and rest.
	start := testModule.New(t)
	separator := testModule.New(t)
	end := testModule.New(t)

	original.AssertNotStarted("on construction of split modules")
	testBar.Run(start, rest, separator, first, end)
	original.AssertStarted("on stream of split modules")

	start.AssertStarted()
	start.Output(outputs.Text("*"))
	separator.AssertStarted()
	separator.Output(outputs.Text("*"))
	end.AssertStarted()
	end.Output(outputs.Text("*"))
	testBar.LatestOutput(0, 2, 4).Expect("setup")

	original.Output(outputs.Group(
		outputs.Text("a"),
		outputs.Text("b"),
		outputs.Text("c"),
		outputs.Text("d"),
		outputs.Text("e"),
	))
	out := testBar.LatestOutput(1, 3)
	out.AssertText([]string{"*", "c", "d", "e", "*", "a", "b", "*"},
		"Segments split up between the two modules")

	out.At(3).LeftClick()
	original.AssertClicked("clicked on rest")

	out.At(5).LeftClick()
	original.AssertClicked("clicked on first")

	out.At(4).LeftClick()
	original.AssertNotClicked("clicked on other")

	original.Output(outputs.Textf("test"))
	testBar.LatestOutput(1, 3).AssertText(
		[]string{"*", "*", "test", "*"},
		"not enough segments for rest")

	original.Output(nil)
	testBar.LatestOutput(1, 3).AssertText(
		[]string{"*", "*", "*"},
		"no segments in original output")

	clickChan := make(chan string, 1)
	original.Output(outputs.Group(
		outputs.Text("a").OnClick(func(bar.Event) { clickChan <- "a" }),
		outputs.Text("b").OnClick(func(bar.Event) { clickChan <- "b" }),
		outputs.Text("c").OnClick(func(bar.Event) { clickChan <- "c" }),
		outputs.Text("d").OnClick(func(bar.Event) { clickChan <- "d" }),
		outputs.Text("e").OnClick(func(bar.Event) { clickChan <- "e" }),
	))

	out = testBar.LatestOutput(1, 3)
	out.At(1).LeftClick()
	require.Equal(t, "c", <-clickChan)

	out.At(6).LeftClick()
	require.Equal(t, "b", <-clickChan)
}
