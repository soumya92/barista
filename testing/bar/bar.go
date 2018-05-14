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
	"encoding/json"
	"io"
	"sync/atomic"
	"time"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/colors"
	"github.com/soumya92/barista/testing/mockio"
	"github.com/soumya92/barista/testing/output"
	"github.com/soumya92/barista/timing"
)

// TestBar represents a "test" barista instance that runs on mockio streams.
// It provides methods to collect the output from any modules added to it.
type TestBar struct {
	assert.TestingT
	stdin        *mockio.Readable
	stdout       *mockio.Writable
	eventEncoder *json.Encoder
	names        []string
}

var instance atomic.Value // of TestBar

// New creates a new TestBar. This must be called before any modules
// are constructed, to ensure globals like timing.NewScheduler() are
// associated with the test instance.
func New(t assert.TestingT) {
	b := &TestBar{
		TestingT: t,
		stdin:    mockio.Stdin(),
		stdout:   mockio.Stdout(),
	}
	b.eventEncoder = json.NewEncoder(b.stdin)
	instance.Store(b)
	barista.TestMode(b.stdin, b.stdout)
	timing.TestMode()
}

// Run starts the TestBar with the given modules.
func Run(m ...bar.Module) {
	go barista.Run(m...)
	b := instance.Load().(*TestBar)
	// consume header and opening '['
	b.stdout.ReadUntil('}', time.Second)
	b.stdout.ReadUntil('[', time.Second)
	// Start the event stream
	b.stdin.WriteString("[")
}

// Time to wait for events that are expected. Overridden in tests.
var positiveTimeout = time.Second

// Time to wait for events that are not expected.
var negativeTimeout = 10 * time.Millisecond

// Time to wait when repeatedly polling output stream before
// assuming the stream is finished.
var pollingTimeout = 20 * time.Millisecond

func (t *TestBar) readJSONOutput(timeout time.Duration) (out string, err error) {
	out, err = t.stdout.ReadUntil(']', timeout)
	if err != nil {
		return
	}
	_, err = t.stdout.ReadUntil(',', timeout)
	t.stdout.ReadUntil('\n', timeout)
	return
}

// outputFromSegments creates a bar.Output from a slice of bar.Segments.
type outputFromSegments []bar.Segment

func (o outputFromSegments) Segments() []bar.Segment {
	return o
}

func parseOutput(jsonStr string) (names []string, output bar.Output, err error) {
	var jsonOutputs []map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &jsonOutputs)
	if err != nil {
		return
	}
	var segments []bar.Segment
	for _, i3map := range jsonOutputs {
		var s bar.Segment
		text, _ := i3map["full_text"].(string)
		if markup, ok := i3map["markup"]; ok && markup.(string) == "pango" {
			s = bar.PangoSegment(text)
		} else {
			s = bar.TextSegment(text)
		}
		if shortText, ok := i3map["short_text"]; ok {
			s.ShortText(shortText.(string))
		}
		if color, ok := i3map["color"]; ok {
			s.Color(colors.Hex(color.(string)))
		}
		if background, ok := i3map["background"]; ok {
			s.Background(colors.Hex(background.(string)))
		}
		if border, ok := i3map["border"]; ok {
			s.Border(colors.Hex(border.(string)))
		}
		if minWidth, ok := i3map["min_width"]; ok {
			switch w := minWidth.(type) {
			case float64:
				s.MinWidth(int(w))
			case string:
				s.MinWidthPlaceholder(w)
			}
		}
		if align, ok := i3map["align"]; ok {
			s.Align(bar.TextAlignment(align.(string)))
		}
		if id, ok := i3map["instance"]; ok {
			s.Identifier(id.(string))
		}
		if urgent, ok := i3map["urgent"]; ok {
			s.Urgent(urgent.(bool))
		}
		if sep, ok := i3map["separator"]; ok {
			s.Separator(sep.(bool))
		}
		if padding, ok := i3map["separator_block_width"]; ok {
			s.Padding(int(padding.(float64)))
		}
		names = append(names, i3map["name"].(string))
		segments = append(segments, s)
	}
	output = outputFromSegments(segments)
	return
}

// AssertNoOutput asserts that the bar did not output anything.
func AssertNoOutput(args ...interface{}) {
	t := instance.Load().(*TestBar)
	if t.stdout.WaitForWrite(negativeTimeout) {
		assert.Fail(t, "Expected no output", args...)
	}
}

// NextOutput returns output assertions for the next output by the bar.
func NextOutput() output.Assertions {
	t := instance.Load().(*TestBar)
	json, err := t.readJSONOutput(positiveTimeout)
	if err != nil {
		assert.Fail(t, "Error in next output", "Failed to read: %s", err)
		return output.New(t, nil)
	}
	var out bar.Output
	t.names, out, err = parseOutput(json)
	if err != nil {
		assert.Fail(t, "Error in next output", "Failed to parse: %s", err)
	}
	return output.New(t, out)
}

// LatestOutput drains any buffered outputs from the bar, and returns
// output assertions for the last output.
func LatestOutput() output.Assertions {
	t := instance.Load().(*TestBar)
	var json string
	var err error
	for err == nil {
		var out string
		out, err = t.readJSONOutput(pollingTimeout)
		if err == nil {
			json = out
		}
	}
	if err != io.EOF {
		// This should never happen, since mockio is backed by a bytes.Buffer,
		// which can only return nil or EOF errors on read.
		assert.Fail(t, "Error in latest output", "Failed to read: %s", err)
		return output.New(t, nil)
	}
	var out bar.Output
	t.names, out, err = parseOutput(json)
	if err != nil {
		assert.Fail(t, "Error in latest output", "Failed to parse: %s", err)
	}
	return output.New(t, out)
}

// SendEvent sends a bar.Event to the segment at position i.
// Important: Events are dispatched based on the segments last read.
// Call LatestOutput or NextOutput to ensure the segment <-> module
// mapping is up to date.
func SendEvent(i int, e bar.Event) {
	t := instance.Load().(*TestBar)
	if i >= len(t.names) {
		assert.Fail(t, "Cannot send event",
			"Clicked on segment %d, but only have %d",
			i, len(t.names))
		return
	}
	t.eventEncoder.Encode(struct {
		bar.Event
		Name string `json:"name"`
	}{
		Event: e,
		Name:  t.names[i],
	})
	t.stdin.WriteString(",\n")
}

// Click sends a left click to the segment at position i.
func Click(i int) {
	SendEvent(i, bar.Event{Button: bar.ButtonLeft})
}

// Tick calls timing.NextTick() under the covers, allowing
// some tests that don't need fine grained scheduling control
// to treat timing's test mode as an implementation detail.
func Tick() {
	timing.NextTick()
}
