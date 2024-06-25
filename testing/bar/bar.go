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

// Package bar provides utilities for testing barista modules
// using a fake bar instance.
package bar

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/core"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/oauth"
	"github.com/soumya92/barista/testing/output"
	"github.com/soumya92/barista/timing"

	"github.com/stretchr/testify/require"
)

// TestBar represents a minimal wrapper around core.ModuleSet that
// simulates a bar for testing purposes.
type TestBar struct {
	require.TestingT
	moduleSet *core.ModuleSet
	outputs   chan testOutput
}

var instance atomic.Value // of TestBar
var encryptionKeySet sync.Once

// New creates a new TestBar. This must be called before any modules
// are constructed, to ensure globals like timing.NewScheduler() are
// associated with the test instance.
func New(t require.TestingT) {
	b := &TestBar{
		TestingT: t,
		outputs:  make(chan testOutput, 10),
	}
	instance.Store(b)
	timing.TestMode()
	encryptionKeySet.Do(func() {
		oauth.SetEncryptionKey([]byte(`not-an-encryption-key`))
	})
}

func debugOut(segments bar.Segments) (texts []string) {
	for _, s := range segments {
		txt, _ := s.Content()
		texts = append(texts, txt)
	}
	return texts
}

// Run starts the TestBar with the given modules.
func Run(m ...bar.Module) {
	b := instance.Load().(*TestBar)
	b.moduleSet = core.NewModuleSet(m)
	go func(b *TestBar) {
		for updated := range b.moduleSet.Stream() {
			segments := make(bar.Segments, 0)
			out := b.moduleSet.LastOutputs()
			for _, mod := range out {
				for _, seg := range mod {
					segments = append(segments, seg)
				}
			}
			l.Fine("%s new output (by %d): %v",
				l.ID(b.moduleSet), updated, debugOut(out[updated]))
			b.outputs <- testOutput{segments, updated}
		}
	}(b)
}

// Time to wait for events that are expected. Overridden in tests.
var positiveTimeout = 10 * time.Second

// Time to wait for events that are not expected.
var negativeTimeout = 10 * time.Millisecond

// testOutput groups related information about the latest output.
type testOutput struct {
	segments bar.Segments
	updated  int
}

// AssertNoOutput asserts that the bar did not output anything.
func AssertNoOutput(args ...interface{}) {
	t := instance.Load().(*TestBar)
	select {
	case <-t.outputs:
		require.Fail(t, "Expected no output", args...)
	case <-time.After(negativeTimeout):
		// test passed.
	}
}

// NextOutput returns output assertions for the next output by the bar.
func NextOutput(formatAndArgs ...interface{}) output.Assertions {
	t := instance.Load().(*TestBar)
	var segments bar.Segments
	select {
	case out := <-t.outputs:
		segments = out.segments
	case <-time.After(positiveTimeout):
		require.Fail(t, "Expected an output, got none", formatAndArgs...)
	}
	return output.New(t, segments)
}

// LatestOutput waits for an output from each of the module indices
// provided, and returns output assertions for the latest output.
// If no indices are provided, it waits for outputs from all modules.
// To wait for any module instead, use NextOutput().
func LatestOutput(indices ...int) output.Assertions {
	deadline := time.After(positiveTimeout)
	updated := map[int]bool{}
	t := instance.Load().(*TestBar)
	if len(indices) == 0 {
		for i := 0; i < t.moduleSet.Len(); i++ {
			indices = append(indices, i)
		}
	}
	l.Fine("%s waiting for output from modules %v", l.ID(t.moduleSet), indices)
	var segments bar.Segments
	for segments == nil {
		select {
		case out := <-t.outputs:
			updated[out.updated] = true
			if hasAllUpdates(updated, indices) {
				segments = out.segments
				l.Fine("%s got output from %v: %v",
					l.ID(t.moduleSet), indices, debugOut(segments))
			}
		case <-deadline:
			missing := []int{}
			for _, i := range indices {
				if !updated[i] {
					missing = append(missing, i)
				}
			}
			require.Fail(t, "Did not receive expected updates",
				"modules %v did not update", missing)
		}
	}
	return output.New(t, segments)
}

// Drain drains all outputs until the given deadline elapses. Can be useful for
// working around coalescing issues where the exact number of outputs can vary
// based on goroutine scheduling. At least one output is expected, and the last
// output received before the deadline is provided for further assertions.
func Drain(wait time.Duration, formatAndArgs ...interface{}) output.Assertions {
	deadline := time.After(wait)
	t := instance.Load().(*TestBar)
	l.Fine("%s waiting for %v", l.ID(t.moduleSet), wait)

	var segments bar.Segments
	hasOutput := false
	waiting := true
	for waiting {
		select {
		case out := <-t.outputs:
			segments = out.segments
			hasOutput = true
			l.Fine("%s got output: %v", l.ID(t.moduleSet), debugOut(segments))
		case <-deadline:
			waiting = false
		}
	}
	if !hasOutput {
		require.Fail(t, "Expected at least one output", formatAndArgs...)
	}
	return output.New(t, segments)
}

func hasAllUpdates(updated map[int]bool, indices []int) bool {
	for _, i := range indices {
		if !updated[i] {
			return false
		}
	}
	return true
}

// Tick calls timing.NextTick() under the covers, allowing
// some tests that don't need fine grained scheduling control
// to treat timing's test mode as an implementation detail.
func Tick() time.Time {
	return timing.NextTick()
}
