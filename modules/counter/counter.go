// Copyright 2017 Google Inc.
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

// Package counter demonstrates an extremely simple i3bar module that shows a counter
// which can be chnaged by clicking on it. It showcases the asynchronous nature of
// i3bar modules when written in go.
package counter

import (
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/outputs"
)

type module struct {
	*base.Base
	count  int
	format string
}

// New constructs a new counter module.
func New(format string) bar.Module {
	m := &module{
		Base:   base.New(),
		count:  0,
		format: format,
	}
	m.Output(outputs.Textf(format, 0))
	return m
}

func (m *module) Click(e bar.Event) {
	switch e.Button {
	case bar.ButtonLeft, bar.ScrollDown, bar.ScrollLeft, bar.ButtonBack:
		m.count--
	case bar.ButtonRight, bar.ScrollUp, bar.ScrollRight, bar.ButtonForward:
		m.count++
	}
	m.Output(outputs.Textf(m.format, m.count))
	m.Base.Click(e)
}
