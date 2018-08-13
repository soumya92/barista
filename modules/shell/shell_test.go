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

package shell

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
)

func TestTail(t *testing.T) {
	testBar.New(t)
	tail := Tail("bash", "-c", "for i in `seq 1 5`; do echo $i; sleep 0.075; done")
	testBar.Run(tail)

	for _, i := range []string{"1", "2", "3", "4", "5"} {
		testBar.NextOutput().AssertText([]string{i}, i)
	}

	testBar.AssertNoOutput("when command terminates normally")

	testBar.New(t)
	tail = Tail("bash", "-c", "for i in `seq 1 3`; do echo $i; sleep 0.075; done; exit 1")
	testBar.Run(tail)
	for _, i := range []string{"1", "2", "3"} {
		testBar.NextOutput().AssertText([]string{i}, i)
	}

	testBar.NextOutput().AssertError(
		"when command terminates with an error")

	testBar.New(t)
	tail = Tail("this-is-not-a-valid-command", "--but", "'have'", "-some", "args")
	testBar.Run(tail)
	testBar.NextOutput().AssertError(
		"when starting an invalid command")
}

func TestEvery(t *testing.T) {
	testBar.New(t)

	rep := Every(time.Second, "echo", "foo")
	testBar.Run(rep)

	testBar.NextOutput().AssertText([]string{"foo"}, "on start")

	then := timing.Now()
	now := timing.NextTick()
	require.InDelta(t, float64(time.Second), float64(now.Sub(then)),
		float64(time.Millisecond))

	testBar.NextOutput().Expect("on tick")
	testBar.AssertNoOutput("until tick")

	for i := 0; i < 10; i++ {
		testBar.Tick()
		testBar.NextOutput().Expect("on tick")
	}

	testBar.New(t)
	rep = Every(time.Second, "this-is-not-a-valid-command", "foo")
	testBar.Run(rep)
	testBar.NextOutput().AssertError("when starting an invalid command")
	testBar.Tick()
	testBar.NextOutput().AssertError("new error output on next tick")
}

func TestOnce(t *testing.T) {
	testBar.New(t)
	testBar.Run(Once("echo", "bar"))
	testBar.NextOutput().AssertText([]string{"bar"}, "on start")
	testBar.AssertNoOutput("after the first output")

	testBar.New(t)
	testBar.Run(Once("this-is-not-a-valid-command", "foo"))
	testBar.NextOutput().AssertError("when starting an invalid command")
	testBar.AssertNoOutput("after the first output")
}

func TestOnClick(t *testing.T) {
	testBar.New(t)
	module := OnClick("echo", "foo")
	m, onFinish := testBar.AddFinishListener(module)
	testBar.Run(m)
	testBar.NextOutput().AssertText([]string{"foo"}, "on start")
	testBar.AssertNoOutput("after the first output")
	<-onFinish
	testBar.Click(0)
	testBar.NextOutput().Expect("click causes restart")
	testBar.NextOutput().AssertText([]string{"foo"}, "on click")
	testBar.AssertNoOutput("after the next output")
}
