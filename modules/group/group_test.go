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

package group

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
)

type simpleModule chan bar.Output

func (s simpleModule) Stream() <-chan bar.Output { return (<-chan bar.Output)(s) }

type clickableModule chan bar.Output

func (c clickableModule) Stream() <-chan bar.Output { return (<-chan bar.Output)(c) }
func (c clickableModule) Click(e bar.Event)         {}

func TestWrappedModule(t *testing.T) {
	evt := bar.Event{X: 1, Y: 1}
	for _, m := range []bar.Module{
		make(simpleModule),
		make(clickableModule),
	} {
		var wrapped WrappedModule = &module{Module: m}
		wrapped.Stream()
		assert.NotPanics(t, func() { wrapped.Click(evt) })
	}
}
