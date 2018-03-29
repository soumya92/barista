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

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/base/scheduler"
	"github.com/soumya92/barista/outputs"
	testModule "github.com/soumya92/barista/testing/module"
)

func TestTail(t *testing.T) {
	tail := Tail("seq", "1", "5")
	tester := testModule.NewOutputTester(t, tail)

	for _, i := range []string{"1", "2", "3", "4", "5"} {
		out := tester.AssertOutput(i)
		assert.Equal(t, out, outputs.Text(i))
	}

	tester.AssertNoOutput("when command terminates normally")

	tail = Tail("bash", "-c", "seq 1 3; exit 1")
	tester = testModule.NewOutputTester(t, tail)
	for _, i := range []string{"1", "2", "3"} {
		out := tester.AssertOutput(i)
		assert.Equal(t, outputs.Text(i), out)
	}

	tester.AssertError("when command terminates with an error")

	tail = Tail("this-is-not-a-valid-command", "--but", "'have'", "-some", "args")
	tester = testModule.NewOutputTester(t, tail)
	tester.AssertError("when starting an invalid command")
}

func TestEvery(t *testing.T) {
	scheduler.TestMode(true)

	rep := Every(time.Second, "echo", "foo")
	tester := testModule.NewOutputTester(t, rep)

	out := tester.AssertOutput("on start")
	assert.Equal(t, outputs.Text("foo"), out)

	then := scheduler.Now()
	now := scheduler.NextTick()
	assert.InDelta(t, time.Second, now.Sub(then), float64(time.Millisecond))

	tester.AssertOutput("on tick")
	tester.AssertNoOutput("until tick")

	for i := 0; i < 10; i++ {
		scheduler.NextTick()
		tester.AssertOutput("on tick")
	}

	rep = Every(time.Second, "this-is-not-a-valid-command", "foo")
	tester = testModule.NewOutputTester(t, rep)
	tester.AssertError("when starting an invalid command")
}

func TestOnce(t *testing.T) {
	tester := testModule.NewOutputTester(t, Once("echo", "bar"))
	out := tester.AssertOutput("on start")
	assert.Equal(t, outputs.Text("bar"), out)
	tester.AssertNoOutput("after the first output")

	tester = testModule.NewOutputTester(t, Once("this-is-not-a-valid-command", "foo"))
	tester.AssertError("when starting an invalid command")
	tester.AssertNoOutput("after the first output")
}
