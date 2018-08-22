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

	"github.com/martinlindhe/unit"
	testBar "github.com/soumya92/barista/testing/bar"
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

	testBar.NextOutput().AssertText(
		[]string{"22.2℃ chance of meatballs (FLDSMDFR)"}, "on start")

	testBar.Tick()
	testBar.NextOutput().Expect("on tick")

	w.Template(`{{.Temperature.Fahrenheit | printf "%.0f"}}, by {{.Attribution}}`)
	testBar.NextOutput().AssertText([]string{
		"72, by FLDSMDFR"}, "on template change")

	p.Lock()
	p.cached = true
	p.Unlock()

	testBar.Tick()
	testBar.AssertNoOutput("on tick when weather is cached")

	p.Lock()
	p.cached = false
	p.error = errors.New("foo")
	p.Unlock()

	testBar.Tick()
	testBar.NextOutput().AssertError("on tick with error")
}
