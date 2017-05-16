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

// NewSegment creates a new output segment with text content.
func NewSegment(text string) Segment {
	return Segment{"full_text": text}
}

// ShortText sets the shortened text, used if the default text
// for all segments does not fit in the bar.
func (s Segment) ShortText(shortText string) Segment {
	s["short_text"] = shortText
	return s
}

// Color sets the foreground color for the segment.
func (s Segment) Color(color Color) Segment {
	return s.setColorValue("color", color)
}

// Background sets the background color for the segment.
func (s Segment) Background(background Color) Segment {
	return s.setColorValue("background", background)
}

// Border sets the border color for the segment.
func (s Segment) Border(border Color) Segment {
	return s.setColorValue("border", border)
}

// Color values are special case, in that the "empty" color value should
// be treated the same as unset so that i3bar treats empty color values
// as its default.
func (s Segment) setColorValue(name string, value Color) Segment {
	if value == "" {
		delete(s, name)
	} else {
		s[name] = value
	}
	return s
}

// MinWidth sets the minimum width for the segment.
func (s Segment) MinWidth(minWidth int) Segment {
	s["min_width"] = minWidth
	return s
}

// Align sets the text alignment within the segment.
func (s Segment) Align(align TextAlignment) Segment {
	s["align"] = align
	return s
}

// Urgent sets the urgency of the segment.
func (s Segment) Urgent(urgent bool) Segment {
	s["urgent"] = urgent
	return s
}

// Separator controls whether this Segment has a separator.
func (s Segment) Separator(separator bool) Segment {
	s["separator"] = separator
	return s
}

// SeparatorWidth sets the width of the separator "block" for the segment.
func (s Segment) SeparatorWidth(separatorWidth int) Segment {
	s["separator_block_width"] = separatorWidth
	return s
}

// Markup sets the markup type (pango or none) for the segment.
func (s Segment) Markup(markup Markup) Segment {
	s["markup"] = markup
	return s
}

// Instance sets the opaque instance name for this Segment.
// Click events on the segment will return the same instance string.
func (s Segment) Instance(instance string) Segment {
	s["instance"] = instance
	return s
}
