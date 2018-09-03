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
	"barista.run/bar"
	"barista.run/base/notifier"
	"barista.run/base/sink"
	l "barista.run/logging"
)

// Sink is a specialisation of bar.Sink that provides bar.Segments
// instead of bar.Outputs, allowing range without nil checks.
type Sink = func(bar.Segments)

// Module represents a bar.Module wrapped with core barista functionality.
// It is used as a building block for the main bar, modules that manipulate
// other modules (group, reformat), and for writing tests.
// It handles restarting the wrapped module on a left/right/middle click,
// as well as providing an option to "replay" the last output from the module.
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
func (m *Module) Stream(sink Sink) {
	for {
		m.runLoop(sink)
	}
}

// runLoop is one iteration of the wrapped module. It starts the wrapped
// module, and multiplexes events, replay notifications, and module output.
// It returns when the underlying module is ready to be restarted (i.e. it
// was stopped and an eligible click event was received).
func (m *Module) runLoop(realSink Sink) {
	started := false
	finished := false
	outputCh, innerSink := sink.New()
	doneCh := make(chan struct{})

	go func(m bar.Module, innerSink bar.Sink, doneCh chan<- struct{}) {
		l.Fine("%s started", l.ID(m))
		m.Stream(innerSink)
		l.Fine("%s finished", l.ID(m))
		doneCh <- struct{}{}
	}(m.original, innerSink, doneCh)

	var out bar.Segments
	for {
		select {
		case o := <-outputCh:
			started = true
			out = toSegments(o)
			realSink(out)
		case <-doneCh:
			finished = true
			l.Fine("%s: set restart handlers", l.ID(m))
			realSink(addRestartHandlers(out, m.restartFn))
		case <-m.replayCh:
			if started {
				l.Fine("%s: replay last output", l.ID(m))
				realSink(out)
			}
		case <-m.restartCh:
			if finished {
				l.Fine("%s restarted", l.ID(m.original))
				realSink(stripErrors(out, l.ID(m)))
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

// toSegments creates a copy of the bar output as bar.Segments.
// This means that nil checks are no longer needed, since bar.Segments
// is just a slice, and nil slice will not cause panics. It also means
// that implementations of bar.Output that have a large backing data
// structure can be gc'd, since only their output segments will be
// stored here.
func toSegments(out bar.Output) bar.Segments {
	if out == nil {
		return nil
	}
	var segs bar.Segments
	for _, s := range out.Segments() {
		segs = append(segs, s.Clone())
	}
	return segs
}

// stripErrors strips any error segments from the given list.
func stripErrors(in bar.Segments, logCtx string) bar.Segments {
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
func addRestartHandlers(in bar.Segments, restartFn func()) bar.Segments {
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
