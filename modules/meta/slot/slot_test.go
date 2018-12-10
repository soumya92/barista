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

package slot

import (
	"testing"

	"barista.run/modules/static"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	testModule "barista.run/testing/module"
)

func TestReformat(t *testing.T) {
	testBar.New(t)
	original := testModule.New(t)
	slotter := New(original)
	original.AssertNotStarted("on construction of slotter module")

	slotA := slotter.Slot("A")
	slotB := slotter.Slot("B")
	slotC := slotter.Slot("C")
	testBar.Run(
		static.New(outputs.Text("<")),
		slotA,
		static.New(outputs.Text("-")),
		slotB,
		static.New(outputs.Text("-")),
		slotC,
		static.New(outputs.Text(">")),
	)

	testBar.LatestOutput(0, 2, 4, 6).AssertText([]string{"<", "-", "-", ">"})
	original.AssertStarted("on stream of slots")

	original.OutputText("foo")
	testBar.AssertNoOutput("not showing in any slot")

	slotter.Activate("A")
	testBar.NextOutput("On slot change").AssertText(
		[]string{"<", "foo", "-", "-", ">"})

	slotter.Activate("B")
	testBar.LatestOutput(1, 3).AssertText(
		[]string{"<", "-", "foo", "-", ">"})

	slotter.Activate("C")
	out := testBar.LatestOutput(3, 5)
	out.AssertText([]string{"<", "-", "-", "foo", ">"})

	out.At(3).LeftClick()
	original.AssertClicked("on click of active slot")

	original.OutputText("new text")
	testBar.NextOutput().AssertText([]string{"<", "-", "-", "new text", ">"})

	original.Output(nil)
	testBar.NextOutput().AssertText([]string{"<", "-", "-", ">"})

	slotter.Activate("A")
	testBar.LatestOutput(1, 5).AssertText([]string{"<", "-", "-", ">"})

	original.OutputText("baz")
	testBar.NextOutput().AssertText([]string{"<", "baz", "-", "-", ">"})
}
