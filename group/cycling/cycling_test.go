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

package cycling

import (
	"testing"
	"time"

	testBar "barista.run/testing/bar"
	testModule "barista.run/testing/module"
	"barista.run/timing"

	"github.com/stretchr/testify/require"
)

func TestCycling(t *testing.T) {
	testBar.New(t)

	tm0 := testModule.New(t)
	tm1 := testModule.New(t)
	tm2 := testModule.New(t)

	grp, ctrl := Group(time.Second, tm0, tm1, tm2)
	tm0.AssertNotStarted("on group creation")
	tm1.AssertNotStarted()
	tm2.AssertNotStarted()

	testBar.Run(grp)
	tm0.AssertStarted("on stream")
	tm1.AssertStarted()
	tm2.AssertStarted()

	start := timing.Now()

	testBar.NextOutput().AssertEmpty("With no module output")
	tm0.OutputText("a")
	testBar.NextOutput().AssertText([]string{"a"},
		"on active module update")
	require.Equal(t, start, timing.Now(), "First output is immediate")

	testBar.Tick()
	testBar.NextOutput().AssertEmpty(
		"switched to module with no output")
	require.Equal(t, start.Add(time.Second), timing.Now())

	tm1.OutputText("b")
	testBar.NextOutput().AssertText([]string{"b"},
		"on active module update")
	require.Equal(t, start.Add(time.Second), timing.Now())

	tm2.OutputText("c")
	testBar.AssertNoOutput("inactive module update")
	require.Equal(t, start.Add(time.Second), timing.Now())

	ctrl.SetInterval(time.Minute)
	testBar.Tick()
	testBar.NextOutput().AssertText([]string{"c"},
		"switched to module with an update")
	require.Equal(t, start.Add(61*time.Second), timing.Now())
}
