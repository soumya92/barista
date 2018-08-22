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

package collapsing

import (
	"testing"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"
	"github.com/stretchr/testify/require"
)

func TestCollapsing(t *testing.T) {
	testBar.New(t)

	tm0 := testModule.New(t)
	tm1 := testModule.New(t)
	tm2 := testModule.New(t)

	grp, ctrl := Group(tm0, tm1, tm2)
	tm0.AssertNotStarted("on group creation")
	tm1.AssertNotStarted()
	tm2.AssertNotStarted()

	testBar.Run(grp)
	tm0.AssertStarted("on stream")
	tm1.AssertStarted()
	tm2.AssertStarted()

	tm0.OutputText("a")
	out := testBar.NextOutput()
	out.AssertText([]string{"+"}, "starts collapsed")

	tm1.OutputText("b")
	testBar.AssertNoOutput("while collapsed")
	require.False(t, ctrl.Expanded())

	out.At(0).LeftClick()
	testBar.NextOutput().AssertText([]string{">", "a", "b", "<"},
		"Expands on click, uses previous output")

	require.True(t, ctrl.Expanded())
	tm2.OutputText("c")
	testBar.NextOutput().AssertText([]string{">", "a", "b", "c", "<"},
		"Updates immediately when expanded")

	ctrl.Collapse()
	testBar.NextOutput().AssertText([]string{"+"})

	ctrl.Collapse()
	testBar.AssertNoOutput("no change when already collapsed")

	ctrl.Expand()
	out = testBar.NextOutput()
	out.AssertText([]string{">", "a", "b", "c", "<"},
		"Uses last output when re-expanded")

	out.At(2).LeftClick()
	tm1.AssertClicked("after expansion")

	out.At(4).LeftClick()
	testBar.NextOutput().AssertText([]string{"+"})
	require.False(t, ctrl.Expanded())

	ctrl.ButtonFunc(func(expanded bool) (start, end bar.Output) {
		if expanded {
			return outputs.Text("->"), outputs.Text("<-")
		}
		return outputs.Text("<->"), nil
	})
	testBar.NextOutput().AssertText([]string{"<->"},
		"On button func change")

	ctrl.Toggle()
	testBar.NextOutput().AssertText([]string{"->", "a", "b", "c", "<-"},
		"On expansion with custom button func")
}
