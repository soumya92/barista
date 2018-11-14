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
	"sort"
	"time"

	"barista.run/bar"
	"barista.run/timing"
)

// Repeat creates a TimedOutput from a function by repeatedly calling it at
// different times.
type Repeat func(time.Time) bar.Output

// Every repeats the output at a fixed interval.
func (r Repeat) Every(interval time.Duration) bar.TimedOutput {
	return &repeating{r, repeatEvery{interval, timing.Now()}}
}

// AtNext repeats the output at multiples of the given duration. e.g.
// AtNext(time.Minute) will repeat the output at 13:00:00, 13:01:00, and so on,
// regardless of the first output time.
func (r Repeat) AtNext(interval time.Duration) bar.TimedOutput {
	return &repeating{r, repeatAtNext{interval}}
}

// At repeats the output at the specified fixed points in time.
func (r Repeat) At(times ...time.Time) bar.TimedOutput {
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})
	return &repeating{r, repeatAtTimes{times}}
}

type timer interface {
	after(time.Time) time.Time
	before(time.Time) time.Time
}

type repeatEvery struct {
	interval time.Duration
	start    time.Time
}

func (r repeatEvery) after(now time.Time) time.Time {
	count := now.Sub(r.start) / r.interval
	return r.start.Add((count + 1) * r.interval)
}

func (r repeatEvery) before(now time.Time) time.Time {
	count := now.Sub(r.start) / r.interval
	return r.start.Add(count * r.interval)
}

type repeatAtNext struct {
	interval time.Duration
}

func (r repeatAtNext) after(now time.Time) time.Time {
	return now.Add(r.interval + 1).Truncate(r.interval)
}

func (r repeatAtNext) before(now time.Time) time.Time {
	return now.Truncate(r.interval)
}

type repeatAtTimes struct {
	times []time.Time
}

func (r repeatAtTimes) after(now time.Time) time.Time {
	for i, t := range r.times {
		if t.After(now) {
			r.times = r.times[i:]
			return t
		}
	}
	return time.Time{}
}

func (r repeatAtTimes) before(now time.Time) time.Time {
	var result time.Time
	for _, t := range r.times {
		if t.After(now) {
			break
		}
		result = t
	}
	return result
}

type repeating struct {
	outputFunc func(time.Time) bar.Output
	timer
}

func (r *repeating) Segments() []*bar.Segment {
	t := r.before(timing.Now())
	if t.IsZero() {
		return nil
	}
	o := r.outputFunc(t)
	if o == nil {
		return nil
	}
	return o.Segments()
}

func (r *repeating) NextRefresh() time.Time {
	return r.after(timing.Now())
}

// resetStartTime is used by Group to ensure that all timed outputs that repeat
// at a fixed interval start their timers together. Perfectly aligning the start
// times for fixed-interval outputs reduces the total number of refresh events,
// by having a single update where timers overlap.
func resetStartTime(out bar.Output, start time.Time) bar.Output {
	r, ok := out.(*repeating)
	if !ok {
		return out
	}
	e, ok := r.timer.(repeatEvery)
	if !ok {
		return out
	}
	return &repeating{r.outputFunc, repeatEvery{e.interval, start}}
}
