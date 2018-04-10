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
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestCounter(t *testing.T) {
	ctr := New("C:%d")
	tester := testModule.NewOutputTester(t, ctr)

	tester.AssertOutputEquals(outputs.Text("C:0"), "on start")

	tester.AssertNoOutput("without any interaction")
	ctr.(bar.Pausable).Pause()
	tester.AssertNoOutput("on pause")
	ctr.(bar.Pausable).Resume()
	tester.AssertNoOutput("on resume")

	ctr.(bar.Clickable).Click(bar.Event{Button: bar.ScrollUp})
	tester.AssertOutputEquals(outputs.Text("C:1"), "on click")

	ctr.(bar.Clickable).Click(bar.Event{Button: bar.ScrollDown})
	tester.AssertOutputEquals(outputs.Text("C:0"), "on click")

	ctr.(bar.Clickable).Click(bar.Event{Button: bar.ButtonBack})
	tester.AssertOutputEquals(outputs.Text("C:-1"), "on click")
}
