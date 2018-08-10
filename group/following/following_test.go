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

package following

import (
	"testing"

	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestFollowing(t *testing.T) {
	testBar.New(t)

	tm0 := testModule.New(t)
	tm1 := testModule.New(t)
	tm2 := testModule.New(t)

	grp := Group(tm0, tm1, tm2)
	tm0.AssertNotStarted("on group creation")
	tm1.AssertNotStarted()
	tm2.AssertNotStarted()

	testBar.Run(grp)
	tm0.AssertStarted("on stream")
	tm1.AssertStarted()
	tm2.AssertStarted()

	testBar.NextOutput().AssertEmpty("With no module output")

	tm0.OutputText("a")
	testBar.NextOutput().AssertText([]string{"a"},
		"on module update")

	tm1.OutputText("b")
	testBar.NextOutput().AssertText([]string{"b"})

	testBar.Click(0)
	tm1.AssertClicked("Last module to output is clicked")

	tm2.OutputText("c")
	testBar.NextOutput().AssertText([]string{"c"})
}
