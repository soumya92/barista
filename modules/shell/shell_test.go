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

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/soumya92/barista/timing"
)

func TestRepeating(t *testing.T) {
	testBar.New(t)

	rep := New("echo", "foo").Every(time.Second)
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

	rep.Every(0)
	testBar.Tick()
	testBar.AssertNoOutput("on zero interval")

	testBar.New(t)
	rep = New("this-is-not-a-valid-command", "foo").Every(time.Second)
	testBar.Run(rep)
	testBar.NextOutput().AssertError("when starting an invalid command")
	out := testBar.NextOutput("sets restart handler")
	testBar.Tick()
	testBar.AssertNoOutput("on next tick with error")
	out.At(0).LeftClick()
	testBar.NextOutput("clears error segment")
	testBar.NextOutput().AssertError("when starting an invalid command")
}

func TestUpdate(t *testing.T) {
	testBar.New(t)
	m := New("echo", "bar").Output(func(in string) bar.Output {
		return outputs.Textf(">%s<", in)
	})
	testBar.Run(m)
	testBar.NextOutput().AssertText([]string{">bar<"}, "on start")
	testBar.AssertNoOutput("after the first output")

	m.Output(func(in string) bar.Output {
		return outputs.Textf("*%s*", in)
	})
	testBar.NextOutput("on format change").AssertText([]string{"*bar*"})

	m.Refresh()
	testBar.NextOutput("on refresh").AssertText([]string{"*bar*"})
}
