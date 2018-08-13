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

package bar

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/testing/fail"
	"github.com/soumya92/barista/testing/module"
	"github.com/soumya92/barista/timing"
)

func TestOutput(t *testing.T) {
	New(t)
	m := module.New(t)
	Run(m)

	m.AssertStarted()
	AssertNoOutput("When module has no output")

	m.OutputText("foo")
	NextOutput().AssertText([]string{"foo"},
		"When module outputs")

	m.Output(pango.Text("bold").Bold())
	NextOutput().AssertText(
		[]string{"<span weight='bold'>bold</span>"},
		"Pango text is passed through")

	s := bar.TextSegment("foo")
	s.Color(colors.Hex("#f00"))
	s.Background(colors.Hex("#ff0"))
	s.Border(colors.Hex("#070"))
	s.MinWidthPlaceholder("##.###")
	s.Urgent(true)
	s.Padding(10)
	s.Separator(false)
	s.Identifier("some-id")
	s.Align(bar.AlignStart)

	m.Output(s)
	NextOutput().AssertEqual(s, "complex segment")

	s.MinWidth(150)
	m.Output(s)
	require.Equal(t, s, NextOutput().At(0).Segment())

	m.OutputText("baz")
	NextOutput().Expect("when output")

	m.Output(nil)
	NextOutput().AssertEmpty("on empty output")
}

func TestEvents(t *testing.T) {
	New(t)
	m1 := module.New(t)
	m2 := module.New(t)
	Run(m1, m2)

	m1.AssertStarted()
	m1.OutputText("1")

	m2.AssertStarted()
	m2.OutputText("2")

	// LatestOutput sets up the segment position <-> module mappings.
	LatestOutput().Expect("on update")

	Click(0)
	m1.AssertClicked("When test bar clicks module")
	m2.AssertNotClicked("When a different module is clicked")

	e := bar.Event{X: 10, Y: 10}
	SendEvent(1, e)
	actual := m2.AssertClicked()
	require.Equal(t, e, actual, "event properties pass through")

	m1.Output(nil)
	m2.Output(outputs.Group(
		outputs.Text("a").Identifier("foo"),
		outputs.Text("b").Identifier("bar"),
		outputs.Text("c").Identifier("baz"),
	))
	LatestOutput(0, 1).Expect("on update")
	Click(0)
	m1.AssertNotClicked("when module has no output")
	evt := m2.AssertClicked("events based on output positions")
	require.Equal(t, "foo", evt.SegmentID, "SegmentID is propagated")
	Click(1)
	evt = m2.AssertClicked("multiple segments from the same module")
	require.Equal(t, "bar", evt.SegmentID)
	Click(2)
	evt = m2.AssertClicked()
	require.Equal(t, "baz", evt.SegmentID)
}

func TestRestartingModule(t *testing.T) {
	New(t)
	m := module.New(t)
	Run(m)

	m.AssertStarted()
	m.Output(outputs.Errorf("something went wrong"))
	m.Close()
	errStrs := NextOutput().AssertError("on error")
	require.Equal(t, []string{"something went wrong"}, errStrs)

	require.Panics(t, func() { m.OutputText("bar") },
		"module is not streaming")

	// Exited with an error, so left click will restart,
	Click(0)
	// and clear the error'd segment.
	NextOutput().AssertText([]string{})

	m.AssertStarted()
	require.NotPanics(t, func() { m.OutputText("baz") },
		"module has restarted")
	NextOutput().AssertText([]string{"baz"})
}

func TestFinishListener(t *testing.T) {
	New(t)
	actual := module.New(t)
	m, listener := AddFinishListener(actual)
	Run(m)
	select {
	case <-listener:
		require.Fail(t, "FinishListener spuriously notified")
	case <-time.After(10 * time.Millisecond):
	}

	actual.AssertStarted()
	actual.Output(outputs.Text("foo"))
	NextOutput().AssertText([]string{"foo"})

	actual.Close()
	select {
	case <-listener:
		// test passed.
	case <-time.After(time.Second):
		require.Fail(t, "FinishListener not notified")
	}

	Click(0)
	NextOutput().AssertText([]string{"foo"})

	actual.AssertStarted()
	select {
	case <-listener:
		require.Fail(t, "FinishListener spuriously notified")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestSegment(t *testing.T) {
	New(t)
	m := module.New(t)
	Run(m)

	m.AssertStarted()
	m.Output(outputs.Group(
		outputs.Text("a"),
		outputs.Text("b"),
		outputs.Errorf("oops"),
	))

	out := NextOutput()
	out.At(0).AssertText("a")
	out.At(1).AssertEqual(bar.TextSegment("b"))
	errStr := out.At(2).AssertError()
	require.Equal(t, "oops", errStr)

	s := bar.PangoSegment("<b>bold</b>").Urgent(true)
	m.Output(s)
	require.Equal(t, s, NextOutput().At(0).Segment())
}

func TestTick(t *testing.T) {
	New(t)
	Run()
	startTime := timing.Now()
	timing.NewScheduler().Every(time.Minute)

	require.Equal(t, startTime.Add(time.Minute), Tick())

	New(t)
	Run()
	newStartTime := timing.Now()
	require.Equal(t, newStartTime, Tick())
}

func assertFails(t *testing.T, testFunc func(*module.TestModule), args ...interface{}) {
	positiveTimeout = 10 * time.Millisecond
	defer func() { positiveTimeout = time.Second }()
	m := module.New(t)

	fail.Setup(func(fakeT *testing.T) {
		New(fakeT)
		Run(m)
		m.AssertStarted()
	}).AssertFails(t, func(*testing.T) {
		testFunc(m)
	}, args...)
}

func TestNoOutput(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		m.OutputText("test")
		AssertNoOutput("with output")
	}, "Asserting no output with output")
}

func TestEventError(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		m.OutputText("test")
		Click(1)
	}, "Clicking on segment out of range")
}

func TestRestartingFailure(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		m.OutputText("foobar")
		RestartModule(0)
	})
}

func TestOutputErrors(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		NextOutput().AssertEqual(outputs.Text("something"))
	}, "Next output when nothing updates")

	assertFails(t, func(m *module.TestModule) {
		LatestOutput(0).AssertText([]string{"abcd"})
	}, "Latest output when nothing updates")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		LatestOutput(1).AssertText([]string{"abcd"})
	}, "Latest output for different index")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		NextOutput().AssertText([]string{"efgh"})
	}, "Output with wrong text value")

	assertFails(t, func(m *module.TestModule) {
		NextOutput().AssertError()
	}, "Asserting error without any output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("1234")
		NextOutput().AssertError()
	}, "Asserting error with non-error output")

	assertFails(t, func(m *module.TestModule) {
		m.Output(nil)
		NextOutput().AssertError()
	}, "Asserting error with empty output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("5678")
		NextOutput().AssertEmpty()
	}, "Asserting empty with non-empty output")
}

func TestSegmentErrors(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		m.OutputText("something")
		NextOutput().At(0).AssertEqual(outputs.Text("else"))
	}, "Next output segment when text doesn't match")

	assertFails(t, func(m *module.TestModule) {
		NextOutput().At(0).AssertEqual(outputs.Text("else"))
	}, "Next output segment without any output")

	assertFails(t, func(m *module.TestModule) {
		LatestOutput(0).At(0).AssertText("abcd")
	}, "Latest output segment when nothing updates")

	assertFails(t, func(m *module.TestModule) {
		NextOutput().At(0).AssertError()
	}, "Asserting error without any output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("1234")
		NextOutput().At(0).AssertError()
	}, "Asserting error with non-error output")

	assertFails(t, func(m *module.TestModule) {
		m.Output(outputs.Group(
			outputs.Text("1234"),
			outputs.Text("5678"),
		))
		NextOutput().At(2).Segment()
	}, "out of range segment")
}
