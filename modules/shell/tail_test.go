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

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
)

func TestTail(t *testing.T) {
	testBar.New(t)
	tail := Tail("bash", "-c", "for i in `seq 1 5`; do echo $i; sleep 0.075; done")
	testBar.Run(tail)

	for _, i := range []string{"1", "2", "3", "4", "5"} {
		testBar.NextOutput().AssertText([]string{i}, i)
	}

	testBar.AssertNoOutput("when command terminates normally")

	tail.Output(func(in string) bar.Output {
		return outputs.Textf("++%s++", in)
	})
	testBar.NextOutput("on format func change").AssertText(
		[]string{"++5++"}, "applies format func to last output line")

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
