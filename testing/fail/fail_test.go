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

package fail

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var failures = []struct {
	fn   func(*testing.T)
	desc string
}{
	{func(t *testing.T) { t.Fail() }, "Fail"},
	{func(t *testing.T) { t.FailNow() }, "FailNow"},
	{func(t *testing.T) { t.Fatal("fatal") }, "Fatal"},
	{func(t *testing.T) { assert.Fail(t, "something") }, "assert.Fail"},
	{func(t *testing.T) { assert.FailNow(t, "error") }, "assert.FailNow"},
	{func(t *testing.T) { require.Fail(t, "required") }, "require.Fail"},
	{func(t *testing.T) { assert.True(t, false) }, "assert.True"},
	{func(t *testing.T) { require.Equal(t, "a", "b") }, "require.Equal"},
}

var noFailures = []struct {
	fn   func(*testing.T)
	desc string
}{
	{func(t *testing.T) {}, "Nop"},
	{func(t *testing.T) { t.Log("everything is awesome") }, "Log"},
	{func(t *testing.T) { require.True(t, true) }, "require.True"},
	{func(t *testing.T) { assert.Equal(t, 4, 2+2) }, "assert.Equal"},
}

func TestFailures(t *testing.T) {
	for _, f := range failures {
		AssertFails(t, f.fn, f.desc)
	}
}

func TestNoFailure(t *testing.T) {
	for _, f := range noFailures {
		assert.False(t, Failed(f.fn), f.desc)
	}
}

func TestAssertionWithNoFailure(t *testing.T) {
	for _, f := range noFailures {
		AssertFails(t, func(t *testing.T) {
			AssertFails(t, f.fn)
		}, f.desc)
	}
}

func TestSetupFailures(t *testing.T) {
	for _, f := range failures {
		for _, ff := range failures {
			AssertFails(t, func(t *testing.T) {
				Setup(f.fn).AssertFails(t, ff.fn,
					"Setup(%s).AssertFails(%s)", f.desc, ff.desc)
			})
		}
		for _, nf := range noFailures {
			AssertFails(t, func(t *testing.T) {
				Setup(f.fn).AssertFails(t, nf.fn,
					"Setup(%s).AssertFails(%s)", f.desc, nf.desc)
			})
		}
	}
}

func TestNoFailureWithSetup(t *testing.T) {
	for _, s := range noFailures {
		for _, nf := range noFailures {
			AssertFails(t, func(t *testing.T) {
				Setup(s.fn).AssertFails(t, nf.fn,
					"Setup(%s).AssertFails(%s)", s.desc, nf.desc)
			})
		}
	}
}

func TestFailureWithSetup(t *testing.T) {
	for _, s := range noFailures {
		for _, f := range failures {
			Setup(s.fn).AssertFails(t, f.fn,
				"Setup(%s).AssertFails(%s)", s.desc, f.desc)
		}
	}
}
