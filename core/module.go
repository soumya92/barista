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

// Package core provides some of the basic barista functionality,
// enabling more complex constructs without duplicating code or logic.
package core

import (
	"sync"
	"time"

	"barista.run/bar"
	"barista.run/base/notifier"
	l "barista.run/logging"
	"barista.run/timing"
)

// Module represents a bar.Module wrapped with core barista functionality.
// It is used as a building block for the main bar, modules that manipulate
// other modules (group, reformat), and for writing tests.
// It handles restarting the wrapped module on a left/right/middle click,
// as well as providing an option to "replay" the last output from the module.
// It also provides timed output functionality.
type Module struct {
	original  bar.Module
	replayCh  <-chan struct{}
	replayFn  func()
	restartCh <-chan struct{}
	restartFn func()
}

// NewModule wraps an existing bar.Module with core barista functionality,
// such as restarts and the ability to replay the last output.
func NewModule(original bar.Module) *Module {
	m := &Module{original: original}
	m.replayFn, m.replayCh = notifier.New()
	m.restartFn, m.restartCh = notifier.New()
	l.Attach(original, m, "~core")
	l.Register(m, "replayCh")
	l.Register(m, "restartCh")
	return m
}

// Stream runs the module with the given sink, automatically handling
// terminations/restarts of the wrapped module.
func (m *Module) Stream(sink bar.Sink) {
	for {
		m.runLoop(sink)
	}
}

// runLoop is one iteration of the wrapped module. It starts the wrapped
// module, and multiplexes events, replay notifications, and module output.
// It returns when the underlying module is ready to be restarted (i.e. it
// was stopped and an eligible click event was received).
func (m *Module) runLoop(realSink bar.Sink) {
	started := false
	finished := false
	var refreshFn func()
	if r, ok := m.original.(bar.RefresherModule); ok {
		refreshFn = r.Refresh
	}
	timedSink := newTimedSink(realSink, refreshFn)
	l.Attach(m.original, timedSink, "~internal-sink")
	outputCh := make(chan bar.Output)
	innerSink := func(o bar.Output) { outputCh <- o }
	doneCh := make(chan struct{})

	go func(m bar.Module, innerSink bar.Sink, doneCh chan<- struct{}) {
		l.Fine("%s started", l.ID(m))
		m.Stream(innerSink)
		l.Fine("%s finished", l.ID(m))
		doneCh <- struct{}{}
	}(m.original, innerSink, doneCh)

	var out bar.Output
	for {
		select {
		case out = <-outputCh:
			started = true
			timedSink.Output(out, true)
		case <-doneCh:
			finished = true
			timedSink.Stop()
			out = toSegments(out)
			l.Fine("%s: set restart handlers", l.ID(m))
			timedSink.Output(addRestartHandlers(out, m.restartFn), false)
		case <-m.replayCh:
			if started {
				l.Fine("%s: replay last output", l.ID(m))
				timedSink.Output(out, true)
			}
		case <-m.restartCh:
			if finished {
				l.Fine("%s restarted", l.ID(m.original))
				timedSink.Output(stripErrors(out, l.ID(m)), false)
				return // Stream will restart the run loop.
			}
		}
	}
}

// Replay sends the last output from the wrapped module to the sink.
func (m *Module) Replay() {
	m.replayFn()
}

// isRestartableClick checks whether a click event should restart the
// wrapped module. A left/right/middle click will restart the module.
func isRestartableClick(e bar.Event) bool {
	return e.Button == bar.ButtonLeft ||
		e.Button == bar.ButtonRight ||
		e.Button == bar.ButtonMiddle
}

// stripErrors strips any error segments from the given list.
func stripErrors(o bar.Output, logCtx string) bar.Segments {
	in := toSegments(o)
	var out bar.Segments
	for _, s := range in {
		if s.GetError() == nil {
			out = append(out, s)
		}
	}
	if len(in) != len(out) {
		l.Fine("%s removed %d error segments from output",
			logCtx, len(in)-len(out))
	}
	return out
}

// addRestartHandlers replaces all click handlers with a function
// that restarts the module. This is used on the last output of
// the wrapped module after the original finishes.
func addRestartHandlers(o bar.Output, restartFn func()) bar.Segments {
	in := toSegments(o)
	var out bar.Segments
	for _, s := range in {
		out = append(out, s.Clone().OnClick(func(e bar.Event) {
			if isRestartableClick(e) {
				restartFn()
			}
		}))
	}
	return out
}

// addRefreshHandlers adds middle-click refresh to the output.
func addRefreshHandlers(o bar.Output, refreshFn func()) bar.Segments {
	in := toSegments(o)
	if refreshFn == nil {
		return in
	}
	var out bar.Segments
	for _, s := range in {
		handleClick := s.Click
		hasError := s.GetError() != nil
		out = append(out, s.Clone().OnClick(func(e bar.Event) {
			switch {
			case e.Button == bar.ButtonMiddle:
				refreshFn()
			case hasError && isRestartableClick(e):
				refreshFn()
			default:
				handleClick(e)
			}
		}))
	}
	return out
}

func toSegments(o bar.Output) bar.Segments {
	if o == nil {
		return nil
	}
	return o.Segments()
}

type staticTimedOutput struct {
	bar.Output
}

func (s staticTimedOutput) Segments() []*bar.Segment {
	return toSegments(s.Output)
}

func (s staticTimedOutput) NextRefresh() time.Time {
	return time.Time{}
}

// timedSink is a wrapper around bar.Sink that supports timed output. It takes
// a single bar.TimedOutput and unrolls it into multiple calls to the underlying
// sink, automatically resetting future calls on new output.
type timedSink struct {
	bar.Sink
	*timing.Scheduler
	refreshFn func()

	mu          sync.Mutex
	out         bar.TimedOutput
	refreshable bool
}

func newTimedSink(original bar.Sink, refreshFn func()) *timedSink {
	t := &timedSink{
		Sink:      original,
		Scheduler: timing.NewScheduler(),
		refreshFn: refreshFn,
	}
	l.Register(t, "Sink", "Scheduler")
	go t.runLoop()
	return t
}

func (t *timedSink) Output(o bar.Output, refreshable bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.refreshable = refreshable
	var ok bool
	t.out, ok = o.(bar.TimedOutput)
	if !ok {
		t.out = staticTimedOutput{o}
		l.Fine("%s: regular output", l.ID(t))
	}
	t.renderLocked()
}

func (t *timedSink) runLoop() {
	for t.Tick() {
		t.render()
	}
}

func (t *timedSink) render() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.renderLocked()
}

func (t *timedSink) renderLocked() {
	if t.out == nil {
		return
	}
	var o bar.Output = t.out
	if next := t.out.NextRefresh(); !next.IsZero() {
		l.Fine("%s: timed output, next refresh %v", l.ID(t), next)
		t.At(next)
	} else {
		t.Stop()
		t.out = nil
	}
	if t.refreshable {
		o = addRefreshHandlers(o, t.refreshFn)
	}
	t.Sink.Output(o)
}
