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

package group

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/outputs"
	testBar "github.com/soumya92/barista/testing/bar"
	testModule "github.com/soumya92/barista/testing/module"

	"github.com/stretchr/testify/require"
)

type simpleGrouper struct {
	visible    []int
	start, end *bar.Segment
	clicked    chan string
}

func (s *simpleGrouper) Visible(index int) bool {
	for _, i := range s.visible {
		if index == i {
			return true
		}
	}
	return false
}

func (s *simpleGrouper) Buttons() (st, en bar.Output) {
	return s.start.OnClick(func(bar.Event) { s.clicked <- "start" }),
		s.end.OnClick(func(bar.Event) { s.clicked <- "end" })
}

func TestSimpleGrouper(t *testing.T) {
	testBar.New(t)

	m0 := testModule.New(t)
	m1 := testModule.New(t)
	m2 := testModule.New(t)
	g := &simpleGrouper{
		visible: []int{0, 2},
		start:   outputs.Text("start"),
		end:     outputs.Text("end"),
		clicked: make(chan string, 10),
	}

	grp := New(g, m0, m1, m2)
	m0.AssertNotStarted("On group construction")
	m1.AssertNotStarted()
	m2.AssertNotStarted()

	testBar.Run(grp)
	m0.AssertStarted("On group stream")
	m1.AssertStarted()
	m2.AssertStarted()

	testBar.NextOutput().AssertText(
		[]string{"start", "end"}, "buttons are shown immediately")

	m1.OutputText("foo")
	testBar.AssertNoOutput("when an visible module updates")

	m0.Output(nil)
	testBar.NextOutput().AssertText([]string{"start", "end"})

	m2.OutputText("foo")
	out := testBar.NextOutput()
	out.AssertText([]string{"start", "foo", "end"})

	out.At(1).Click(bar.Event{})
	m2.AssertClicked("clicks pass through the group")

	out.At(0).Click(bar.Event{})
	require.Equal(t, "start", <-g.clicked)

	out.At(2).Click(bar.Event{})
	require.Equal(t, "end", <-g.clicked)
}

type lockableGrouper struct {
	*testing.T
	*simpleGrouper
	lockCount, unlockCount int64
}

func (l *lockableGrouper) Lock() {
	atomic.AddInt64(&l.lockCount, 1)
}

func (l *lockableGrouper) Unlock() {
	atomic.AddInt64(&l.unlockCount, 1)
}

func (l *lockableGrouper) getCounts() (locks, unlocks int) {
	locks = int(atomic.LoadInt64(&l.lockCount))
	unlocks = int(atomic.LoadInt64(&l.unlockCount))
	return
}

func (l *lockableGrouper) assertLocked(method string) {
	locks, unlocks := l.getCounts()
	if locks-unlocks != 1 {
		require.Fail(l.T, "Called "+method+" without lock")
	}
}

func (l *lockableGrouper) Visible(index int) bool {
	l.assertLocked("Visible")
	return l.simpleGrouper.Visible(index)
}

func (l *lockableGrouper) Buttons() (s, e bar.Output) {
	l.assertLocked("Buttons")
	return l.simpleGrouper.Buttons()
}

func TestLockableGrouper(t *testing.T) {
	testBar.New(t)

	m0 := testModule.New(t)
	m1 := testModule.New(t)
	m2 := testModule.New(t)
	g := &lockableGrouper{
		T: t,
		simpleGrouper: &simpleGrouper{
			visible: []int{0, 2},
			start:   outputs.Text("start"),
			end:     outputs.Text("end"),
			clicked: make(chan string, 10),
		}}

	grp := New(g, m0, m1, m2)
	m0.AssertNotStarted("On group construction")
	locks, unlocks := g.getCounts()
	require.Equal(t, 0, locks+unlocks, "No locks on start")

	testBar.Run(grp)
	m0.AssertStarted("On group stream")
	m1.AssertStarted()
	m2.AssertStarted()
	testBar.NextOutput().AssertText([]string{"start", "end"})
	locks, unlocks = g.getCounts()
	require.Equal(t, 1, locks, "Initial output locks")
	require.Equal(t, 1, unlocks, "Equal unlock count")

	m0.OutputText("foo")
	out := testBar.NextOutput()
	out.AssertText([]string{"start", "foo", "end"})
	locks, unlocks = g.getCounts()
	require.Equal(t, 2, locks, "One lock per output")
	require.Equal(t, 2, unlocks, "Equal unlock count")

	m1.OutputText("baz")
	testBar.AssertNoOutput("when hidden module updates")
	locks, unlocks = g.getCounts()
	require.Equal(t, 3, locks, "Locks even if visible returns false")
	require.Equal(t, 3, unlocks)

	out.At(0).Click(bar.Event{})
	locks, unlocks = g.getCounts()
	require.Equal(t, 3, locks, "Does not lock for button click")

	out.At(1).Click(bar.Event{})
	locks, unlocks = g.getCounts()
	require.Equal(t, 3, locks, "Does not lock for module click")
	require.Equal(t, 3, unlocks)
}

type signallingGrouper struct {
	*simpleGrouper
	notifyCh chan struct{}
}

func (s *signallingGrouper) Signal() <-chan struct{} {
	return s.notifyCh
}

func TestSignallingGrouper(t *testing.T) {
	testBar.New(t)

	m0 := testModule.New(t)
	m1 := testModule.New(t)
	m2 := testModule.New(t)
	g := &signallingGrouper{
		simpleGrouper: &simpleGrouper{
			visible: []int{0, 2},
			start:   outputs.Text("start"),
			end:     outputs.Text("end"),
			clicked: make(chan string, 10),
		},
		notifyCh: make(chan struct{}),
	}

	grp := New(g, m0, m1, m2)

	testBar.Run(grp)
	m0.AssertStarted("On group stream")
	m1.AssertStarted()
	m2.AssertStarted()
	testBar.NextOutput().AssertText([]string{"start", "end"})

	m0.OutputText("foo")
	testBar.NextOutput().AssertText([]string{"start", "foo", "end"})

	m1.OutputText("baz")
	testBar.AssertNoOutput("when hidden module updates")

	g.notifyCh <- struct{}{}
	testBar.NextOutput().AssertText([]string{"start", "foo", "end"})

	m2.OutputText("test")
	testBar.NextOutput().AssertText([]string{"start", "foo", "test", "end"})

	g.visible = []int{1}
	g.notifyCh <- struct{}{}
	testBar.NextOutput().AssertText([]string{"start", "baz", "end"})
}

type updatingGrouper struct {
	*simpleGrouper
	updated   map[int]chan bool
	visibleCh chan []int
}

func (u *updatingGrouper) Updated(idx int) {
	u.updated[idx] <- true
	select {
	case u.visible = <-u.visibleCh:
	default:
	}
}

func (u *updatingGrouper) AssertUpdated(t *testing.T, idx int, formatAndArgs ...interface{}) {
	select {
	case <-u.updated[idx]:
		// test passed.
	case <-time.After(time.Second):
		require.Fail(t, "Expected an updated", formatAndArgs...)
	}
}

func TestUpdatingGrouper(t *testing.T) {
	testBar.New(t)

	m0 := testModule.New(t)
	m1 := testModule.New(t)
	m2 := testModule.New(t)
	g := &updatingGrouper{
		simpleGrouper: &simpleGrouper{
			visible: []int{0, 2},
			start:   outputs.Text("start"),
			end:     outputs.Text("end"),
			clicked: make(chan string, 10),
		},
		updated: map[int]chan bool{
			0: make(chan bool),
			1: make(chan bool),
			2: make(chan bool),
		},
		visibleCh: make(chan []int, 1),
	}

	grp := New(g, m0, m1, m2)

	testBar.Run(grp)
	m0.AssertStarted("On group stream")
	m1.AssertStarted()
	m2.AssertStarted()
	testBar.NextOutput().AssertText([]string{"start", "end"})

	m0.OutputText("foo")
	g.AssertUpdated(t, 0, "updated called when module updates")
	testBar.NextOutput().AssertText([]string{"start", "foo", "end"})

	m1.OutputText("baz")
	g.AssertUpdated(t, 1, "updated called, even for hidden module")
	testBar.AssertNoOutput("when hidden module updates")

	m1.OutputText("test")
	g.visibleCh <- []int{1}
	g.AssertUpdated(t, 1)
	out := testBar.NextOutput()
	out.AssertText([]string{"start", "test", "end"},
		"Visibility is computed after update notification")

	out.At(1).Click(bar.Event{})
	select {
	case <-g.updated[1]:
		require.Fail(t, "Expected no updated", "on click")
	case <-time.After(10 * time.Millisecond):
		// test passed, expected no update.
	}
}

func TestSimple(t *testing.T) {
	testBar.New(t)

	m0 := testModule.New(t)
	m1 := testModule.New(t)
	m2 := testModule.New(t)

	grp := Simple(m0, m1, m2)
	m0.AssertNotStarted("On group construction")
	m1.AssertNotStarted()
	m2.AssertNotStarted()

	testBar.Run(grp)
	m0.AssertStarted("On group stream")
	m1.AssertStarted()
	m2.AssertStarted()

	testBar.NextOutput().AssertEmpty("on start")

	m1.OutputText("foo")
	testBar.NextOutput().AssertText([]string{"foo"})

	m0.Output(nil)
	testBar.NextOutput().AssertText([]string{"foo"})

	m2.OutputText("baz")
	out := testBar.NextOutput()
	out.AssertText([]string{"foo", "baz"})

	out.At(1).Click(bar.Event{})
	m2.AssertClicked("clicks pass through the group")
}
