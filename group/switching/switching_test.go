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

package switching

import (
	"testing"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	testModule "barista.run/testing/module"

	"github.com/stretchr/testify/require"
)

func TestSwitching(t *testing.T) {
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

	require.Equal(t, 3, ctrl.Count())
	require.Equal(t, 0, ctrl.Current())
	out := testBar.NextOutput()
	out.AssertText([]string{">"},
		"with no output from module")

	out.At(0).LeftClick()
	testBar.NextOutput().AssertText([]string{"<", ">"})
	require.Equal(t, 1, ctrl.Current())

	ctrl.Next()
	out = testBar.NextOutput()
	out.AssertText([]string{"<"})

	tm1.OutputText("a")
	testBar.AssertNoOutput("on hidden module update")

	out.At(0).LeftClick()
	testBar.NextOutput().AssertText([]string{"<", "a", ">"})

	ctrl.ButtonFunc(func(current, total int) (start, end bar.Output) {
		return outputs.Text("/*"), outputs.Text("*/")
	})
	testBar.NextOutput().AssertText([]string{"/*", "a", "*/"})

	tm0.OutputText("0")
	testBar.AssertNoOutput("on hidden module update")

	ctrl.Show(0)
	out = testBar.NextOutput()
	out.AssertText([]string{"/*", "0", "*/"})
	out.At(0).LeftClick()
	testBar.NextOutput().AssertText([]string{"/*", "*/"})
	require.Equal(t, 2, ctrl.Current(), "wraparound on left")

	ctrl.Next()
	testBar.NextOutput().AssertText([]string{"/*", "0", "*/"})
	require.Equal(t, 0, ctrl.Current(), "wraparound on right")
}
