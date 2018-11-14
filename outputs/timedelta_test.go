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

func TestTimeDelta(t *testing.T) {
	timing.TestMode()

	fixedPoint := timing.Now().Add(70*time.Hour + 33*time.Minute + 8*time.Second)
	o := AtTimeDelta(func(dur time.Duration) bar.Output {
		return Textf("%v", dur)
	}).From(fixedPoint)

	require.Equal(t, fixedPoint.Add(-70*time.Hour), o.NextRefresh())

	assertNextTexts(t, o, []string{"-70h0m0s"})
	assertNextTexts(t, o, []string{"-69h0m0s"})

	timing.AdvanceBy(30 * time.Minute)
	assertCurrentTexts(t, o, []string{"-69h0m0s"})
	assertNextTexts(t, o, []string{"-68h0m0s"})

	timing.AdvanceBy(67*time.Hour + time.Minute)
	assertCurrentTexts(t, o, []string{"-59m0s"})
	assertNextTexts(t, o, []string{"-58m0s"})

	timing.AdvanceBy(57 * time.Minute)
	assertCurrentTexts(t, o, []string{"-1m0s"})
	assertNextTexts(t, o, []string{"-59s"})
	assertNextTexts(t, o, []string{"-58s"})

	timing.AdvanceBy(time.Minute)
	assertCurrentTexts(t, o, []string{"2s"})
	assertNextTexts(t, o, []string{"3s"})

	timing.AdvanceBy(56 * time.Second)
	assertCurrentTexts(t, o, []string{"59s"})
	assertNextTexts(t, o, []string{"1m0s"})
	assertNextTexts(t, o, []string{"2m0s"})

	timing.AdvanceBy(20 * time.Second)
	assertCurrentTexts(t, o, []string{"2m0s"})
	assertNextTexts(t, o, []string{"3m0s"})

	timing.AdvanceBy(56 * time.Minute)
	assertCurrentTexts(t, o, []string{"59m0s"})
	assertNextTexts(t, o, []string{"1h0m0s"})
	assertNextTexts(t, o, []string{"2h0m0s"})
}

func TestFineTimeDelta(t *testing.T) {
	timing.TestMode()

	fixedPoint := timing.Now().Add(70*time.Hour + 33*time.Minute + 8*time.Second)
	o := AtTimeDelta(func(dur time.Duration) bar.Output {
		return Textf("%v", dur)
	}).FromFine(fixedPoint)

	require.Equal(t, fixedPoint.Add(-70*time.Hour), o.NextRefresh())

	assertNextTexts(t, o, []string{"-70h0m0s"})
	assertNextTexts(t, o, []string{"-69h0m0s"})

	timing.AdvanceBy(43*time.Hour + 20*time.Minute)
	assertCurrentTexts(t, o, []string{"-26h0m0s"})
	assertNextTexts(t, o, []string{"-25h0m0s"})
	assertNextTexts(t, o, []string{"-24h0m0s"})
	assertNextTexts(t, o, []string{"-23h59m0s"})
	assertNextTexts(t, o, []string{"-23h58m0s"})

	timing.AdvanceBy(22*time.Hour + 56*time.Minute)
	assertCurrentTexts(t, o, []string{"-1h2m0s"})
	assertNextTexts(t, o, []string{"-1h1m0s"})
	assertNextTexts(t, o, []string{"-1h0m0s"})
	assertNextTexts(t, o, []string{"-59m59s"})
	assertNextTexts(t, o, []string{"-59m58s"})

	timing.AdvanceBy(time.Hour)
	assertCurrentTexts(t, o, []string{"2s"})
	assertNextTexts(t, o, []string{"3s"})

	timing.AdvanceBy(55 * time.Second)
	assertCurrentTexts(t, o, []string{"58s"})
	assertNextTexts(t, o, []string{"59s"})
	assertNextTexts(t, o, []string{"1m0s"})
	assertNextTexts(t, o, []string{"1m1s"})

	timing.AdvanceBy(time.Hour)
	assertCurrentTexts(t, o, []string{"1h1m0s"})
	assertNextTexts(t, o, []string{"1h2m0s"})
}
