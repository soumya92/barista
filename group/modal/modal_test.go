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

package modal

import (
	"testing"
	"unicode"

	"barista.run/bar"
	"barista.run/colors"
	testBar "barista.run/testing/bar"
	testModule "barista.run/testing/module"
	"barista.run/testing/output"

	colorful "github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/require"
)

func assertColorsOfSwitcher(t *testing.T, switcher output.Assertions,
	start int, colors []string, fmtAndArgs ...interface{}) {
	actual := []string{}
	for i := range colors {
		sc, _ := switcher.At(start + i).Segment().GetBackground()
		scf, _ := colorful.MakeColor(sc)
		actual = append(actual, scf.Hex())
	}
	require.Equal(t, colors, actual, fmtAndArgs...)
}

const inactive = "#ff0000"
const active = "#0000ff"

func TestModal(t *testing.T) {
	testBar.New(t)
	colors.LoadFromMap(map[string]string{
		"inactive_workspace_bg": inactive,
		"focused_workspace_bg":  active,
	})

	m := map[string]*testModule.TestModule{}
	for _, key := range []string{
		"a0", "a1", "A2",
		"b0", "b1",
		"c0",
		"D0",
		"Ee0", "Ee1", "E2", "e3",
		"f0", "F1", "f2",
	} {
		m[key] = testModule.New(t)
	}

	modal := New()
	modal.Mode("a").Detail(m["a0"], m["a1"]).Summary(m["A2"])
	modal.Mode("b").Detail(m["b0"], m["b1"])
	modal.Mode("c").Detail(m["c0"])
	modal.Mode("d").Summary(m["D0"])
	modal.Mode("e").Add(m["Ee0"], m["Ee1"]).Summary(m["E2"]).Detail(m["e3"])
	modal.Mode("f").SetOutput(nil).Detail(m["f0"]).Summary(m["F1"]).Detail(m["f2"])

	mod, ctrl := modal.Build()
	testBar.Run(mod)

	for _, mod := range m {
		mod.AssertStarted("on group start")
	}

	require.Equal(t, []string{"a", "b", "c", "d", "e", "f"}, ctrl.Modes())
	require.Empty(t, ctrl.Current())

	testBar.NextOutput().AssertText([]string{"a", "b", "c", "d", "e"},
		"Mode switching button")

	var latestOut output.Assertions
	for k, v := range m {
		v.OutputText(k)
		if unicode.IsUpper([]rune(k)[0]) {
			latestOut = testBar.NextOutput("on summary module update")
		} else {
			testBar.AssertNoOutput("on detail module update")
		}
	}

	latestOut.AssertText([]string{
		"A2", "D0", "Ee0", "Ee1", "E2", "F1",
		"a", "b", "c", "d", "e",
	})
	assertColorsOfSwitcher(t, latestOut, 6,
		[]string{inactive, inactive, inactive, inactive, inactive},
		"no mode active on start")

	latestOut.At(7).LeftClick()
	latestOut = testBar.NextOutput("on mode switch")

	latestOut.AssertText([]string{"b0", "b1", "a", "b", "c", "d", "e"})
	require.Equal(t, "b", ctrl.Current())
	assertColorsOfSwitcher(t, latestOut, 2,
		[]string{inactive, active, inactive, inactive, inactive},
		"clicked mode marked active in switcher")

	ctrl.Activate("b")
	testBar.AssertNoOutput("on activation of current mode")

	latestOut.At(3).LeftClick()
	latestOut = testBar.NextOutput("on clicking current mode")
	latestOut.AssertText([]string{
		"A2", "D0", "Ee0", "Ee1", "E2", "F1",
		"a", "b", "c", "d", "e",
	}, "resets to no active mode")

	latestOut.At(10).LeftClick()
	latestOut = testBar.NextOutput("On clicking inactive mode")
	latestOut.AssertText(
		[]string{"Ee0", "Ee1", "e3", "a", "b", "c", "d", "e"},
		"summary/detail/both modules are handled properly")
	assertColorsOfSwitcher(t, latestOut, 3,
		[]string{inactive, inactive, inactive, inactive, active})

	ctrl.Activate("f")
	latestOut = testBar.NextOutput("on controller mode activation")
	latestOut.AssertText([]string{"f0", "f2", "a", "b", "c", "d", "e"})
	require.Equal(t, "f", ctrl.Current())
	assertColorsOfSwitcher(t, latestOut, 2,
		[]string{inactive, inactive, inactive, inactive, inactive},
		"when active mode has no output")

	ctrl.SetOutput("f", bar.TextSegment("custom"))
	latestOut = testBar.NextOutput("on mode output change")
	latestOut.AssertText([]string{"f0", "f2", "a", "b", "c", "d", "e", "custom"})
	assertColorsOfSwitcher(t, latestOut, 2,
		[]string{inactive, inactive, inactive, inactive, inactive, active})

	ctrl.SetOutput("b", nil)
	latestOut = testBar.NextOutput("on mode output change")
	latestOut.AssertText([]string{"f0", "f2", "a", "c", "d", "e", "custom"})
	assertColorsOfSwitcher(t, latestOut, 2,
		[]string{inactive, inactive, inactive, inactive, active})

	ctrl.Reset()
	latestOut = testBar.NextOutput("on controller reset")
	latestOut.AssertText([]string{
		"A2", "D0", "Ee0", "Ee1", "E2", "F1",
		"a", "c", "d", "e", "custom",
	}, "resets to no active mode")
}
