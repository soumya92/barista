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

package core

import (
	"testing"
	"time"

	"github.com/soumya92/barista/bar"
	testModule "github.com/soumya92/barista/testing/module"
	"github.com/stretchr/testify/require"
)

func nextUpdate(t *testing.T, ch <-chan int, formatAndArgs ...interface{}) int {
	select {
	case idx := <-ch:
		return idx
	case <-time.After(time.Second):
		require.Fail(t, "No update from moduleset", formatAndArgs...)
	}
	return -1
}

func assertNoUpdate(t *testing.T, ch <-chan int, formatAndArgs ...interface{}) {
	select {
	case <-ch:
		require.Fail(t, "Unexpected update from moduleset", formatAndArgs...)
	case <-time.After(10 * time.Millisecond):
		// test passed.
	}
}

func TestModuleSet(t *testing.T) {
	tms := []*testModule.TestModule{
		testModule.New(t),
		testModule.New(t),
		testModule.New(t),
	}
	ms := NewModuleSet([]bar.Module{tms[0], tms[1], tms[2]})
	updateCh := ms.Stream()
	assertNoUpdate(t, updateCh, "on start")
	for _, tm := range tms {
		tm.AssertStarted("on moduleset stream")
	}
	require.Equal(t, 3, ms.Len())

	tms[1].OutputText("foo")
	require.Equal(t, 1, nextUpdate(t, updateCh, "on output"),
		"update notification on new output from module")

	tms[0].OutputText("baz")
	require.Equal(t, 0, nextUpdate(t, updateCh, "on update"),
		"update notification on new output from module")

	require.Empty(t, ms.LastOutput(2), "without any output")
	txt, _ := ms.LastOutput(0)[0].Content()
	require.Equal(t, "baz", txt)

	out := ms.LastOutputs()
	require.Equal(t, 1, len(out[0]))
	txt, _ = out[0][0].Content()
	require.Equal(t, "baz", txt)
	require.Equal(t, 1, len(out[1]))
	txt, _ = out[1][0].Content()
	require.Equal(t, "foo", txt)
	require.Empty(t, out[2])
}
