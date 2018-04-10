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

package bar_test

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	// pango depends on bar
	. "github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/pango"
)

func TestColorInPango(t *testing.T) {
	assert.Equal(t,
		"<span color='#ff0000'>test</span>",
		pango.Span(Color("#ff0000"), "test").Pango(),
		"bar.Color is recognised as a pango attribute")
}
