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

package static

import (
	"testing"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/stretchr/testify/require"
)

func TestStatic(t *testing.T) {
	testBar.New(t)

	s0 := New(nil)
	s1 := New(outputs.Text("foo"))
	s2 := new(Module)
	testBar.Run(s0, s1, s2)

	testBar.LatestOutput().AssertText([]string{"foo"}, "on start")
	testBar.AssertNoOutput("no change to static modules")

	s0.Set(outputs.Text("baz"))
	testBar.LatestOutput(0).AssertText([]string{"baz", "foo"})

	s1.Clear()
	testBar.LatestOutput(1).AssertText([]string{"baz"})

	s2.Clear()
	testBar.LatestOutput(2).AssertText([]string{"baz"})

	clickCh := make(chan bool, 1)
	s2.Set(outputs.Text("foo").OnClick(func(bar.Event) { clickCh <- true }))

	out := testBar.LatestOutput(2)
	out.AssertText([]string{"baz", "foo"})

	select {
	case <-clickCh:
		require.Fail(t, "spurious click event")
	case <-time.After(10 * time.Millisecond):
		// test passed
	}

	out.At(0).LeftClick()
	select {
	case <-clickCh:
		require.Fail(t, "spurious click event")
	case <-time.After(10 * time.Millisecond):
		// test passed
	}

	out.At(1).LeftClick()
	select {
	case v := <-clickCh:
		require.True(t, v, "click handler triggered")
	case <-time.After(time.Second):
		require.Fail(t, "click event not received")
	}
}
