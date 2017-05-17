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

package bar

// Color sets the color for all segments in the output.
func (o Output) Color(color Color) Output {
	for _, s := range o {
		s.Color(color)
	}
	return o
}

// Background sets the background color for all segments in the output.
func (o Output) Background(background Color) Output {
	for _, s := range o {
		s.Background(background)
	}
	return o
}

// Border sets the border color for all segments in the output.
func (o Output) Border(border Color) Output {
	for _, s := range o {
		s.Border(border)
	}
	return o
}

// MinWidth sets the minimum width for each segment in the output.
func (o Output) MinWidth(minWidth int) Output {
	for _, s := range o {
		s.MinWidth(minWidth)
	}
	return o
}

// Align sets the text alignment for all segments in the output.
func (o Output) Align(align TextAlignment) Output {
	for _, s := range o {
		s.Align(align)
	}
	return o
}

// Urgent sets the urgency flag for all segments in the output.
func (o Output) Urgent(urgent bool) Output {
	for _, s := range o {
		s.Urgent(urgent)
	}
	return o
}

// Markup sets the markup type for all segments in the output.
func (o Output) Markup(markup Markup) Output {
	for _, s := range o {
		s.Markup(markup)
	}
	return o
}
