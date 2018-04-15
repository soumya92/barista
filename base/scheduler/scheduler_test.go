// Copyright 2017 Google Inc.
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

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"
)

type doFunc struct {
	ch chan interface{}
	t  *testing.T
}

func (d doFunc) Func() {
	d.ch <- nil
}

func (d doFunc) assertCalled(message string) {
	select {
	case <-d.ch:
	case <-time.After(time.Second):
		assert.Fail(d.t, "doFunc was not called", message)
	}
}

func (d doFunc) assertNotCalled(message string) {
	select {
	case <-d.ch:
		assert.Fail(d.t, "doFunc was called", message)
	case <-time.After(10 * time.Millisecond):
	}
}

func newDoFunc(t *testing.T) doFunc {
	return doFunc{make(chan interface{}, 5), t}
}

func TestSchedulers(t *testing.T) {
	d := newDoFunc(t)

	sch := Do(d.Func)
	d.assertNotCalled("when not scheduled")

	sch.After(5 * time.Millisecond).Stop()
	d.assertNotCalled("when stopped")

	sch.Every(5 * time.Millisecond).Stop()
	d.assertNotCalled("when stopped")

	sch.At(Now().Add(5 * time.Millisecond)).Stop()
	d.assertNotCalled("when stopped")

	sch.At(Now().Add(5 * time.Millisecond)).Pause()
	d.assertNotCalled("when paused")

	sch.Resume()
	d.assertCalled("when resumed")

	sch.Resume()
	d.assertNotCalled("repeated resume is nop")

	sch.After(10 * time.Millisecond)
	d.assertCalled("after interval elapses")

	sch.Stop()
	d.assertNotCalled("when elapsed scheduler is stopped")

	sch.Stop()
	d.assertNotCalled("when elapsed scheduler is stopped again")

	d2 := newDoFunc(t)
	sch = Do(d2.Func).Every(5 * time.Millisecond)
	d2.assertCalled("after interval elapses")
	d2.assertCalled("after interval elapses")
	d2.assertCalled("after interval elapses")
	sch.Pause()
	d2.assertNotCalled("when paused")
	time.Sleep(15 * time.Millisecond) // > 2 intervals.
	sch.Resume()
	d2.assertCalled("when resumed")
	sch.Stop()
	d2.assertNotCalled("only once on resume")

	sch.Stop()
	d2.assertNotCalled("when stopped")

	sch.After(5 * time.Millisecond)
	d2.assertCalled("after delay elapses")
	d2.assertNotCalled("after first trigger")
}

func TestTestMode(t *testing.T) {
	d1 := newDoFunc(t)
	d2 := newDoFunc(t)
	d3 := newDoFunc(t)

	TestMode(true)

	sch1 := Do(d1.Func)
	sch2 := Do(d2.Func)
	sch3 := Do(d3.Func)

	startTime := Now()
	assert.Equal(t, startTime, NextTick(),
		"next tick doesn't change time when nothing is scheduled")
	d1.assertNotCalled("when not scheduled")
	d2.assertNotCalled("when not scheduled")
	d3.assertNotCalled("when not scheduled")

	sch1.After(time.Hour)
	sch2.After(time.Second)
	sch3.After(time.Minute)

	assert.Equal(t, startTime.Add(time.Second), NextTick(),
		"triggers earliest scheduler")
	d2.assertCalled("triggers earliest scheduler")
	d1.assertNotCalled("only earliest scheduler triggers")
	d3.assertNotCalled("only earliest scheduler triggers")

	assert.Equal(t, startTime.Add(time.Minute), NextTick(),
		"triggers next scheduler")
	d2.assertNotCalled("already elapsed")
	d3.assertCalled("earliest scheduler triggers")
	d1.assertNotCalled("not yet elapsed")

	AdvanceBy(20 * time.Minute)
	d2.assertNotCalled("already elapsed")
	d3.assertNotCalled("already elapsed")
	d1.assertNotCalled("did not advance far enough")

	AdvanceBy(2 * time.Hour)
	d2.assertNotCalled("already elapsed")
	d3.assertNotCalled("already elapsed")
	d1.assertCalled("when advancing beyond trigger duration")

	sch1.Every(time.Minute)
	sch2.Every(10 * time.Minute)
	now := Now()
	for i := 1; i < 10; i++ {
		assert.Equal(t,
			now.Add(time.Duration(i)*time.Minute),
			NextTick(),
			"repeated scheduler")
		d1.assertCalled("repeated scheduler")
	}
	assert.Equal(t,
		now.Add(time.Duration(10)*time.Minute),
		NextTick(), "at overlap")
	d1.assertCalled("at overlap")
	d2.assertCalled("at overlap")

	now = Now()
	sch1.Every(time.Minute)
	sch2.After(time.Minute)
	sch3.At(Now().Add(time.Minute))
	assert.Equal(t, now.Add(time.Minute), NextTick(), "multiple triggers")
	d1.assertCalled("multiple triggers")
	d2.assertCalled("multiple triggers")
	d3.assertCalled("multiple triggers")

	AdvanceBy(59*time.Second + 999*time.Millisecond)
	d1.assertNotCalled("before interval elapses")

	AdvanceBy(10 * time.Millisecond)
	d1.assertCalled("after interval elapses")
}

func TestPauseResumeInTestMode(t *testing.T) {
	d := newDoFunc(t)
	TestMode(true)

	sch := Do(d.Func)

	start := Now()
	sch.Pause()
	sch.Every(time.Minute)
	assert.Equal(t, start.Add(time.Minute), NextTick(), "with paused scheduler")
	d.assertNotCalled("while paused")
	assert.Equal(t, start.Add(2*time.Minute), NextTick(), "with paused scheduler")
	d.assertNotCalled("while paused")
	assert.Equal(t, start.Add(3*time.Minute), NextTick(), "with paused scheduler")
	d.assertNotCalled("while paused")
	AdvanceBy(30 * time.Second)
	d.assertNotCalled("while paused")
	sch.Resume()
	d.assertCalled("when resumed")
	d.assertNotCalled("only once when resume")
	assert.Equal(t, start.Add(4*time.Minute), NextTick(), "with resumed scheduler")
	d.assertCalled("tick after resuming")
}

func TestTestModeReset(t *testing.T) {

	TestMode(true)

	d1 := newDoFunc(t)
	Do(d1.Func).Every(time.Second)

	startTime := Now()
	assert.Equal(t, startTime.Add(time.Second), NextTick())
	d1.assertCalled("triggers every second")

	assert.Equal(t, startTime.Add(2*time.Second), NextTick())
	d1.assertCalled("triggers every second")

	TestMode(true)
	d2 := newDoFunc(t)
	Do(d2.Func).Every(time.Minute)

	startTime = Now()
	assert.Equal(t, startTime.Add(time.Minute), NextTick())
	d1.assertNotCalled("previous scheduler is not triggered")
	d2.assertCalled("new scheduler is triggered")

	assert.Equal(t, startTime.Add(2*time.Minute), NextTick())
	d1.assertNotCalled("previous scheduler is not triggered")
	d2.assertCalled("new scheduler is repeatedly triggered")
}
