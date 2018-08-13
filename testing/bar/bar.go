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
	"sync/atomic"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/core"
	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/notifier"
	"github.com/soumya92/barista/testing/output"
	"github.com/soumya92/barista/timing"
)

// TestBar represents a minimal wrapper around core.ModuleSet that
// simulates a bar for testing purposes.
type TestBar struct {
	require.TestingT
	moduleSet  *core.ModuleSet
	segmentIDs []segmentID
	outputs    chan testOutput
}

var instance atomic.Value // of TestBar

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
}

func debugOut(segments bar.Segments) (texts []string) {
	for _, s := range segments {
		texts = append(texts, s.Text())
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
			ids := make([]segmentID, 0)
			out := b.moduleSet.LastOutputs()
			for index, mod := range out {
				for _, seg := range mod {
					segments = append(segments, seg)
					id, _ := seg.GetID()
					ids = append(ids, segmentID{index, id})
				}
			}
			l.Fine("%s new output (by %d): %v",
				l.ID(b.moduleSet), updated, debugOut(out[updated]))
			b.outputs <- testOutput{segments, ids, updated}
		}
	}(b)
}

// Time to wait for events that are expected. Overridden in tests.
var positiveTimeout = time.Second

// Time to wait for events that are not expected.
var negativeTimeout = 10 * time.Millisecond

// segmentID stores the module index and segment identifier,
// which together identify a segment when dispatching events.
type segmentID struct {
	index      int
	identifier string
}

// testOutput groups related information about the latest output.
type testOutput struct {
	segments bar.Segments
	ids      []segmentID
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
func NextOutput() output.Assertions {
	t := instance.Load().(*TestBar)
	var segments bar.Segments
	select {
	case out := <-t.outputs:
		t.segmentIDs = out.ids
		segments = out.segments
	case <-time.After(positiveTimeout):
		require.Fail(t, "Expected an output, got none")
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
				t.segmentIDs = out.ids
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

func hasAllUpdates(updated map[int]bool, indices []int) bool {
	for _, i := range indices {
		if !updated[i] {
			return false
		}
	}
	return true
}

// SendEvent sends a bar.Event to the segment at position i.
// Important: Events are dispatched based on the segments last read.
// Call LatestOutput or NextOutput to ensure the segment <-> module
// mapping is up to date.
func SendEvent(i int, e bar.Event) {
	t := instance.Load().(*TestBar)
	if i >= len(t.segmentIDs) {
		require.Fail(t, "Cannot send event",
			"Clicked on segment %d, but only have %d",
			i, len(t.segmentIDs))
		return
	}
	e.SegmentID = t.segmentIDs[i].identifier
	t.moduleSet.Click(t.segmentIDs[i].index, e)
}

// Click sends a left click to the segment at position i.
func Click(i int) {
	SendEvent(i, bar.Event{Button: bar.ButtonLeft})
}

// Restart sends a left click and consumes the next output, with the
// assumption that the module at the given index has finished. Since
// restarting a module always causes an update, a single method to
// restart and swallow the update makes test code cleaner.
func RestartModule(i int) {
	Click(i)
	LatestOutput(i).Expect("on restart")
}

// Tick calls timing.NextTick() under the covers, allowing
// some tests that don't need fine grained scheduling control
// to treat timing's test mode as an implementation detail.
func Tick() time.Time {
	return timing.NextTick()
}

// moduleWithFinishListener wraps a bar.Module with a function that
// notifies tests that the module is 'finished', i.e. the next click
// will restart it. This is useful for some synchronisations that are
// otherwise really hard to implement.
type moduleWithFinishListener struct {
	bar.Module
	finished func()
}

func (m *moduleWithFinishListener) ModuleFinished() {
	if f, ok := m.Module.(core.ModuleFinishListener); ok {
		f.ModuleFinished()
	}
	m.finished()
}

// AddFinshListener takes a bar.Module and adds a channel that will
// signal when the module finishes, used for synchronisation in tests.
func AddFinishListener(m bar.Module) (bar.Module, <-chan struct{}) {
	r := &moduleWithFinishListener{Module: m}
	var notifyCh <-chan struct{}
	r.finished, notifyCh = notifier.New()
	return r, notifyCh
}
