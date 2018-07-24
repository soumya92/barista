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

package weather

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/martinlindhe/unit"
	"github.com/soumya92/barista/bar"
	testBar "github.com/soumya92/barista/testing/bar"
	"github.com/stretchr/testify/assert"
)

type testProvider struct {
	sync.RWMutex
	Weather
	error
	cached bool
}

func (t *testProvider) GetWeather() (*Weather, error) {
	t.RLock()
	defer t.RUnlock()
	if t.error != nil || t.cached {
		return nil, t.error
	}
	return &t.Weather, nil
}

func TestWeather(t *testing.T) {
	assert := assert.New(t)
	testBar.New(t)
	p := &testProvider{Weather: Weather{
		Location:    "Swallow Falls",
		Condition:   Cloudy,
		Description: "chance of meatballs",
		Temperature: unit.FromFahrenheit(72),
		Humidity:    0.7,
		Attribution: "FLDSMDFR",
	}}
	w := New(p)
	testBar.Run(w)

	testBar.LatestOutput().AssertText(
		[]string{"22.2℃ chance of meatballs (FLDSMDFR)"}, "on start")
	assert.True(true)

	assert.NotPanics(func() { testBar.Click(0) })
	testBar.Tick()
	testBar.LatestOutput().Expect("on tick")

	clickedWeathers := make(chan Weather)
	w.OnClick(func(w Weather, e bar.Event) {
		clickedWeathers <- w
	})

	select {
	case <-clickedWeathers:
		assert.Fail("Click handler triggered by old click")
	case <-time.After(time.Millisecond):
	}

	p.Lock()
	p.Humidity = 0.9
	p.Unlock()

	testBar.Tick()
	testBar.LatestOutput().Expect("on tick")
	testBar.Click(0)

	select {
	case w := <-clickedWeathers:
		assert.InDelta(0.9, w.Humidity, 1e-9)
	case <-time.After(time.Second):
		assert.Fail("Click event did not trigger handler")
	}

	w.Template(`{{.Temperature.Fahrenheit | printf "%.0f"}}, by {{.Attribution}}`)
	testBar.LatestOutput().AssertText([]string{
		"72, by FLDSMDFR"}, "on template change")

	p.Lock()
	p.cached = true
	p.Unlock()

	testBar.Tick()
	testBar.AssertNoOutput("on tick when weather is cached")
	testBar.Click(0)
	assert.Equal(Cloudy, (<-clickedWeathers).Condition)

	p.Lock()
	p.cached = false
	p.error = errors.New("foo")
	p.Unlock()

	testBar.Tick()
	testBar.LatestOutput().AssertError("on tick with error")
	testBar.Click(0)
	select {
	case <-clickedWeathers:
		assert.Fail("Click handler triggered during error")
	case <-time.After(time.Millisecond):
	}
}
