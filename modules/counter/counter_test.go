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

package counter

import (
	"testing"

	"github.com/soumya92/barista/bar"
	testBar "github.com/soumya92/barista/testing/bar"
)

func TestCounter(t *testing.T) {
	ctr := New("C:%d")
	testBar.New(t)
	testBar.Run(ctr)

	testBar.NextOutput().AssertText(
		[]string{"C:0"}, "on start")

	testBar.AssertNoOutput("without any interaction")

	testBar.SendEvent(0, bar.Event{Button: bar.ScrollUp})
	testBar.NextOutput().AssertText(
		[]string{"C:1"}, "on click")

	testBar.SendEvent(0, bar.Event{Button: bar.ScrollDown})
	testBar.NextOutput().AssertText(
		[]string{"C:0"}, "on click")

	testBar.SendEvent(0, bar.Event{Button: bar.ButtonBack})
	testBar.NextOutput().AssertText(
		[]string{"C:-1"}, "on click")

	ctr.Format("=%d=")
	testBar.NextOutput().AssertText(
		[]string{"=-1="}, "on format change")

	testBar.SendEvent(0, bar.Event{Button: bar.ScrollUp})
	testBar.NextOutput().AssertText(
		[]string{"=0="}, "on click after format change")
}
