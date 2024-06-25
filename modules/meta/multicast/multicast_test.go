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

package multicast

import (
	"testing"

	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestMulticast(t *testing.T) {
	testBar.New(t)

	original := testModule.New(t)
	mcast := New(original)

	original.AssertNotStarted("when wrapped")
	testBar.Run(mcast, mcast, mcast)
	original.AssertStarted("on stream of multicast modules")

	testBar.LatestOutput().AssertEmpty("On start with no output")

	original.OutputText("foo")
	out := testBar.LatestOutput()
	out.AssertText([]string{"foo", "foo", "foo"},
		"All copies update with new output")

	out.At(1).LeftClick()
	original.AssertClicked("clicked on multicasted output")

	original.Output(outputs.Group(
		outputs.Text("test"), outputs.Text("baz"),
	))
	testBar.LatestOutput().AssertText(
		[]string{"test", "baz", "test", "baz", "test", "baz"},
		"multiple segments from original module")

	original.Output(nil)
	testBar.LatestOutput().AssertEmpty("empty output from original")

	original.Output(outputs.Errorf("something went wrong"))
	testBar.LatestOutput().AssertError("On error in original")

	original.Close()
	out = testBar.LatestOutput()
	original.AssertNotStarted()

	out.At(0).LeftClick()
	original.AssertStarted("On click of multicast output")
}
