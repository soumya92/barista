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

package pango

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/bar"
)

func TestBasicAttributes(t *testing.T) {
	out := Span(
		Font("monospace"),
		Size(10),
		Weight(500),
		Alpha(0.5),
		BgAlpha(1.0),
		Rise(-100),
		LetterSpacing(0.5),
	).Pango()
	assert.Equal(t, `<span`+
		` face='monospace'`+
		` size='10240'`+
		` weight='500'`+
		` alpha='32767'`+
		` background_alpha='65535'`+
		` rise='-100'`+
		` letter_spacing='512'`+
		`></span>`, out)
}

func TestKeywordAttributes(t *testing.T) {
	out := Span(
		Small,
		Oblique,
		Bold,
		SmallCaps,
		Condensed,
		UnderlineError,
		NoStrikethrough,
	).Pango()
	assert.Equal(t, `<span`+
		` size='small'`+
		` style='oblique'`+
		` weight='bold'`+
		` variant='smallcaps'`+
		` stretch='condensed'`+
		` underline='error'`+
		` strikethrough='false'`+
		`></span>`, out)
}

func TestColorAttributes(t *testing.T) {
	out := Span(
		bar.Color("red"),
		Background(bar.Color("yellow")),
		UnderlineColor(bar.Color("green")),
		StrikethroughColor(bar.Color("black")),
	).Pango()
	assert.Equal(t, `<span`+
		` color='red'`+
		` background='yellow'`+
		` underline_color='green'`+
		` strikethrough_color='black'`+
		`></span>`, out)
}
