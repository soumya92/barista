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
	"io"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/outputs"
	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/testing/mockio"
	"github.com/soumya92/barista/testing/module"
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

	m.Output(outputs.Pango("bold", pango.Bold))
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
	LatestOutput().AssertEqual(s, "complex segment")

	s.MinWidth(150)
	m.Output(s)
	assert.Equal(t, s, NextOutput().At(0).Segment())

	m.OutputText("baz")
	LatestOutput().Expect("when output")

	m.Output(outputs.Empty())
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
	assert.Equal(t, e, actual, "event properties pass through")

	m1.Output(outputs.Empty())
	m2.Output(outputs.Group(
		outputs.Text("a").Identifier("foo"),
		outputs.Text("b").Identifier("bar"),
		outputs.Text("c").Identifier("baz"),
	))
	LatestOutput().Expect("on update")
	Click(0)
	m1.AssertNotClicked("when module has no output")
	evt := m2.AssertClicked("events based on output positions")
	assert.Equal(t, "foo", evt.SegmentID, "SegmentID is propagated")
	Click(1)
	evt = m2.AssertClicked("multiple segments from the same module")
	assert.Equal(t, "bar", evt.SegmentID)
	Click(2)
	evt = m2.AssertClicked()
	assert.Equal(t, "baz", evt.SegmentID)
}

func TestRestartingModule(t *testing.T) {
	New(t)
	m := module.New(t)
	Run(m)

	m.AssertStarted()
	m.Output(outputs.Errorf("something went wrong"))
	m.Close()
	errStrs := NextOutput().AssertError("on error")
	assert.Equal(t, []string{"something went wrong"}, errStrs)

	assert.Panics(t, func() { m.OutputText("bar") },
		"module is not streaming")

	// Died with an error, so right click will nag,
	RightClick(0)
	m.AssertNotStarted("on right click with error")
	err := AssertNagbar("on right click with error")
	assert.Equal(t, "something went wrong", err)

	// but left click will restart,
	Click(0)
	// and clear the error'd segment.
	NextOutput().AssertText([]string{})

	m.AssertStarted()
	assert.NotPanics(t, func() { m.OutputText("baz") },
		"module has restarted")
	NextOutput().AssertText([]string{"baz"})
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

	out := LatestOutput()
	out.At(0).AssertText("a")
	out.At(1).AssertEqual(bar.TextSegment("b"))
	errStr := out.At(2).AssertError()
	assert.Equal(t, "oops", errStr)

	s := bar.PangoSegment("<b>bold</b>").Urgent(true)
	m.Output(s)
	assert.Equal(t, s, LatestOutput().At(0).Segment())
}

func assertFails(t *testing.T, testFunc func(*module.TestModule), args ...interface{}) {
	positiveTimeout = 10 * time.Millisecond
	defer func() { positiveTimeout = time.Second }()
	fakeT := &testing.T{}

	New(fakeT)
	m := module.New(t)
	Run(m)

	m.AssertStarted()
	assert.False(t, fakeT.Failed())

	testFunc(m)
	assert.True(t, fakeT.Failed(), args...)
}

func stdout() *mockio.Writable {
	return instance.Load().(*TestBar).stdout
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

func TestOutputParsingErrors(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		stdout().Write([]byte("[{x}],\n"))
		NextOutput().Expect("should fail")
	}, "Next output with non-json")

	assertFails(t, func(m *module.TestModule) {
		stdout().Write([]byte("foo}],\n"))
		LatestOutput().Expect("should fail")
	}, "Next output with non-json")
}

func TestOutputErrors(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		NextOutput().AssertEqual(outputs.Text("something"))
	}, "Next output when nothing updates")

	assertFails(t, func(m *module.TestModule) {
		LatestOutput().AssertText([]string{"abcd"})
	}, "Latest output when nothing updates")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		LatestOutput().AssertText([]string{"efgh"})
	}, "Latest output with wrong text value")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		stdout().ShouldError(io.ErrNoProgress)
		LatestOutput().AssertText([]string{"abcd"})
	}, "Output when stdout write fails")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		stdout().ShouldError(io.ErrNoProgress)
		LatestOutput().Expect("an output")
	}, "Output when stdout write fails")

	assertFails(t, func(m *module.TestModule) {
		LatestOutput().AssertError()
	}, "Asserting error without any output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("1234")
		LatestOutput().AssertError()
	}, "Asserting error with non-error output")

	assertFails(t, func(m *module.TestModule) {
		m.Output(outputs.Empty())
		LatestOutput().AssertError()
	}, "Asserting error with empty output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("5678")
		LatestOutput().AssertEmpty()
	}, "Asserting empty with non-empty output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		stdout().ShouldError(io.ErrNoProgress)
		LatestOutput().AssertEmpty()
	}, "AssertEmpty when stdout write fails")
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
		LatestOutput().At(0).AssertText("abcd")
	}, "Latest output segment when nothing updates")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("abcd")
		stdout().ShouldError(io.ErrNoProgress)
		LatestOutput().At(0).AssertText("abcd")
	}, "Segment when stdout write fails")

	assertFails(t, func(m *module.TestModule) {
		LatestOutput().At(0).AssertError()
	}, "Asserting error without any output")

	assertFails(t, func(m *module.TestModule) {
		m.OutputText("1234")
		LatestOutput().At(0).AssertError()
	}, "Asserting error with non-error output")

	var seg bar.Segment
	assertFails(t, func(m *module.TestModule) {
		m.Output(outputs.Group(
			outputs.Text("1234"),
			outputs.Text("5678"),
		))
		seg = LatestOutput().At(2).Segment()
	}, "out of range segment")
	assert.Equal(t, bar.Segment{}, seg,
		"zero value on out of range segment")
}

func TestNagbarError(t *testing.T) {
	assertFails(t, func(m *module.TestModule) {
		m.OutputText("test")
		RightClick(0)
		AssertNagbar("on right-click")
	}, "Asserting nagbar on non-error segment")
}
