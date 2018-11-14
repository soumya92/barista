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

package outputs

import (
	"time"

	"barista.run/bar"
	"barista.run/timing"
)

// AtTimeDelta creates a TimedOutput from a function by repeatedly calling it at
// different times, using a fixed point in time as a reference point.
type AtTimeDelta func(time.Duration) bar.Output

// From sets the reference point and creates a timed output that repeats the
// given function. The repeat rate is:
// - delta < 1 minute: every second
// - delta < 1 hour: every minute
// - otherwise: every hour
// This is useful if the output displays a single time unit (e.g. 3m, or 8h).
func (a AtTimeDelta) From(time time.Time) bar.TimedOutput {
	return &repeatOnDelta{time, a, timeDeltaCoarse}
}

// FromFine is From with more rapid updates:
// - delta < 1 hour: every second
// - delta < 24 hours: every minute
// - otherwise: every hour
// This is useful if the output displays two time units (e.g. 5h3m, or 2d7h).
func (a AtTimeDelta) FromFine(time time.Time) bar.TimedOutput {
	return &repeatOnDelta{time, a, timeDeltaFine}
}

type repeatOnDelta struct {
	time.Time
	outputFunc  func(time.Duration) bar.Output
	granularity func(time.Duration) time.Duration
}

func (r *repeatOnDelta) Segments() []*bar.Segment {
	delta, truncated, granularity := r.durations()
	if truncated > delta && granularity < 0 {
		truncated += granularity
	}
	o := r.outputFunc(truncated)
	if o == nil {
		return nil
	}
	return o.Segments()
}

func (r *repeatOnDelta) NextRefresh() time.Time {
	delta, truncated, granularity := r.durations()
	if truncated <= delta {
		if granularity > 0 {
			truncated += granularity
		} else {
			truncated -= granularity
		}
	}
	return r.Add(truncated)
}

func (r *repeatOnDelta) durations() (delta, truncated, granularity time.Duration) {
	delta = timing.Now().Sub(r.Time)
	if delta > 0 {
		granularity = r.granularity(delta + 1)
	} else {
		granularity = -r.granularity(-delta - 1)
	}
	truncated = delta / granularity * granularity
	return // delta, truncated, granularity
}

func timeDeltaFine(in time.Duration) time.Duration {
	if in <= time.Hour {
		return time.Second
	}
	if in <= 24*time.Hour {
		return time.Minute
	}
	return time.Hour
}

func timeDeltaCoarse(in time.Duration) time.Duration {
	if in <= time.Minute {
		return time.Second
	}
	if in <= time.Hour {
		return time.Minute
	}
	return time.Hour
}
