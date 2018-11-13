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
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/timing"

	"github.com/stretchr/testify/require"
)

func assertCurrentTexts(t *testing.T, o bar.TimedOutput, texts []string, formatAndArgs ...interface{}) {
	actuals := []string{}
	if o != nil {
		for _, s := range o.Segments() {
			txt, _ := s.Content()
			actuals = append(actuals, txt)
		}
	}
	require.Equal(t, texts, actuals, formatAndArgs...)
}

func assertNextTexts(t *testing.T, o bar.TimedOutput, texts []string, formatAndArgs ...interface{}) {
	timing.AdvanceTo(o.NextRefresh())
	assertCurrentTexts(t, o, texts, formatAndArgs...)
}

func TestRepeatEvery(t *testing.T) {
	timing.TestMode()

	o := Repeat(func(now time.Time) bar.Output {
		return Text(now.In(time.UTC).Format("15:04"))
	}).Every(time.Minute)

	start := timing.Now()
	require.Equal(t, start.Add(time.Minute), o.NextRefresh())
	assertCurrentTexts(t, o, []string{"20:47"})

	timing.AdvanceBy(30 * time.Second)
	require.Equal(t, start.Add(time.Minute), o.NextRefresh(),
		"Less than interval, NextRefresh is not affected")

	timing.AdvanceBy(5 * time.Minute)
	require.Equal(t, start.Add(6*time.Minute), o.NextRefresh())
	assertCurrentTexts(t, o, []string{"20:52"})

	assertNextTexts(t, o, []string{"20:53"})
	assertNextTexts(t, o, []string{"20:54"})
	assertNextTexts(t, o, []string{"20:55"})
}

func TestRepeatAtNext(t *testing.T) {
	timing.TestMode()

	o := Repeat(func(now time.Time) bar.Output {
		hr := now.In(time.UTC).Hour()
		if hr < 12 {
			return nil
		}
		return Textf("midnight-%d", 24-hr)
	}).AtNext(time.Hour)

	start := timing.Now().Add(13 * time.Minute) // testmode starts at 20:47 UTC.
	require.Equal(t, start, o.NextRefresh())
	assertCurrentTexts(t, o, []string{"midnight-4"})

	timing.AdvanceBy(5 * time.Hour)
	require.Equal(t, start.Add(5*time.Hour), o.NextRefresh())
	require.Empty(t, o.Segments(), "on nil output from underlying func")

	timing.AdvanceTo(start.Add(13 * time.Hour))
	require.Equal(t, start.Add(14*time.Hour), o.NextRefresh(),
		"now is on exact boundary")

	assertNextTexts(t, o, []string{}, "nil output")
	assertNextTexts(t, o, []string{"midnight-12"}, "after previous nil output")
	assertNextTexts(t, o, []string{"midnight-11"})
	assertNextTexts(t, o, []string{"midnight-10"})

	timing.AdvanceBy(8 * time.Hour)
	assertCurrentTexts(t, o, []string{"midnight-2"})
	assertNextTexts(t, o, []string{"midnight-1"})
	assertNextTexts(t, o, []string{}, "nil output after previous non-nil")

	prev := timing.Now()
	assertNextTexts(t, o, []string{})
	require.Equal(t, prev.Add(time.Hour), timing.Now(),
		"ticker continues even with multiple nil outputs")
}

func TestRepeatAt(t *testing.T) {
	timing.TestMode()

	now := timing.Now()
	o := Repeat(func(now time.Time) bar.Output {
		return Text(now.In(time.UTC).Format("15:04:05"))
	}).At(
		now.Add(time.Minute),
		now.Add(time.Hour),
		now.Add(15*time.Minute),
		now.Add(24*time.Hour),
		now.Add(5*time.Hour+30*time.Minute+22*time.Second),
	)

	assertCurrentTexts(t, o, []string{}, "No text until first time elapsed")

	assertNextTexts(t, o, []string{"20:48:00"})
	timing.AdvanceBy(time.Hour)
	assertNextTexts(t, o, []string{"02:17:22"})
	assertNextTexts(t, o, []string{"20:47:00"})

	require.Empty(t, o.NextRefresh(),
		"Once all times have elapsed")
	assertCurrentTexts(t, o, []string{"20:47:00"},
		"Can get output after last time has elapsed")

	timing.AdvanceBy(time.Hour)
	assertCurrentTexts(t, o, []string{"20:47:00"},
		"Last output persists forever")
	timing.AdvanceBy(500*time.Hour + 49*time.Minute + 39*time.Second)
	assertCurrentTexts(t, o, []string{"20:47:00"})
}

func TestRepeatDelta(t *testing.T) {
	timing.TestMode()

	fixedPoint := timing.Now().Add(70*time.Hour + 33*time.Minute + 8*time.Second)
	o := Repeat(func(now time.Time) bar.Output {
		return Textf("%v", fixedPoint.Sub(now))
	}).DeltaFrom(fixedPoint)

	require.Equal(t, fixedPoint.Add(-70*time.Hour), o.NextRefresh())

	assertNextTexts(t, o, []string{"70h0m0s"})
	assertNextTexts(t, o, []string{"69h0m0s"})

	timing.AdvanceBy(30 * time.Minute)
	assertCurrentTexts(t, o, []string{"69h0m0s"})
	assertNextTexts(t, o, []string{"68h0m0s"})

	timing.AdvanceBy(67*time.Hour + time.Minute)
	assertCurrentTexts(t, o, []string{"59m0s"})
	assertNextTexts(t, o, []string{"58m0s"})

	timing.AdvanceBy(57 * time.Minute)
	assertCurrentTexts(t, o, []string{"1m0s"})
	assertNextTexts(t, o, []string{"59s"})
	assertNextTexts(t, o, []string{"58s"})

	timing.AdvanceBy(time.Minute)
	assertCurrentTexts(t, o, []string{"-2s"})
	assertNextTexts(t, o, []string{"-3s"})

	timing.AdvanceBy(56 * time.Second)
	assertCurrentTexts(t, o, []string{"-59s"})
	assertNextTexts(t, o, []string{"-1m0s"})
	assertNextTexts(t, o, []string{"-2m0s"})

	timing.AdvanceBy(20 * time.Second)
	assertCurrentTexts(t, o, []string{"-2m0s"})
	assertNextTexts(t, o, []string{"-3m0s"})

	timing.AdvanceBy(56 * time.Minute)
	assertCurrentTexts(t, o, []string{"-59m0s"})
	assertNextTexts(t, o, []string{"-1h0m0s"})
	assertNextTexts(t, o, []string{"-2h0m0s"})
}

func TestRepeatFineDelta(t *testing.T) {
	timing.TestMode()

	fixedPoint := timing.Now().Add(70*time.Hour + 33*time.Minute + 8*time.Second)
	o := Repeat(func(now time.Time) bar.Output {
		return Textf("%v", fixedPoint.Sub(now))
	}).FineDeltaFrom(fixedPoint)

	require.Equal(t, fixedPoint.Add(-70*time.Hour), o.NextRefresh())

	assertNextTexts(t, o, []string{"70h0m0s"})
	assertNextTexts(t, o, []string{"69h0m0s"})

	timing.AdvanceBy(43*time.Hour + 20*time.Minute)
	assertCurrentTexts(t, o, []string{"26h0m0s"})
	assertNextTexts(t, o, []string{"25h0m0s"})
	assertNextTexts(t, o, []string{"24h0m0s"})
	assertNextTexts(t, o, []string{"23h59m0s"})
	assertNextTexts(t, o, []string{"23h58m0s"})

	timing.AdvanceBy(22*time.Hour + 56*time.Minute)
	assertCurrentTexts(t, o, []string{"1h2m0s"})
	assertNextTexts(t, o, []string{"1h1m0s"})
	assertNextTexts(t, o, []string{"1h0m0s"})
	assertNextTexts(t, o, []string{"59m59s"})
	assertNextTexts(t, o, []string{"59m58s"})

	timing.AdvanceBy(time.Hour)
	assertCurrentTexts(t, o, []string{"-2s"})
	assertNextTexts(t, o, []string{"-3s"})

	timing.AdvanceBy(55 * time.Second)
	assertCurrentTexts(t, o, []string{"-58s"})
	assertNextTexts(t, o, []string{"-59s"})
	assertNextTexts(t, o, []string{"-1m0s"})
	assertNextTexts(t, o, []string{"-1m1s"})

	timing.AdvanceBy(time.Hour)
	assertCurrentTexts(t, o, []string{"-1h1m0s"})
	assertNextTexts(t, o, []string{"-1h2m0s"})
}
