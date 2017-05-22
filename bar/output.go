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

import "math"

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

/*
Width and separator(width) are treated specially such that the methods
make sense when called on a single-segment output (such as the result
of outputs.Textf(...)) as well as when called on a multi-segment output.

To that end, min-width distributes the minimum width equally amongst
all segments, and separator(width) only operate on the last segment.

Additional methods for "inner" separator(width) operate on all but the
last segment.
*/

// MinWidth sets the minimum width for the output, by (mostly) equally
// distributing the given minWidth amongst all segments in the output.
func (o Output) MinWidth(minWidth int) Output {
	remainingWidth := float64(minWidth)
	for idx, s := range o {
		remainingSegments := float64(len(o) - idx)
		myWidth := math.Floor(remainingWidth/remainingSegments + 0.5)
		s.MinWidth(int(myWidth))
		remainingWidth = remainingWidth - myWidth
	}
	return o
}

// Separator sets the separator visibility of the last segment in the output.
func (o Output) Separator(separator bool) Output {
	if len(o) > 0 {
		o[len(o)-1].Separator(separator)
	}
	return o
}

// SeparatorWidth sets the separator width of the last segment in the output.
func (o Output) SeparatorWidth(separatorWidth int) Output {
	if len(o) > 0 {
		o[len(o)-1].SeparatorWidth(separatorWidth)
	}
	return o
}

// InnerSeparator sets the separator visibility between segments of this output.
func (o Output) InnerSeparator(separator bool) Output {
	for idx, s := range o {
		if idx+1 < len(o) {
			s.Separator(separator)
		}
	}
	return o
}

// InnerSeparatorWidth sets the separator width between segments of this output.
func (o Output) InnerSeparatorWidth(separatorWidth int) Output {
	for idx, s := range o {
		if idx+1 < len(o) {
			s.SeparatorWidth(separatorWidth)
		}
	}
	return o
}

// NewSegment creates a new output segment with text content.
func NewSegment(text string) Segment {
	return Segment{"full_text": text}
}

// Text returns the text content of this segment.
func (s Segment) Text() string {
	return s["full_text"].(string)
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
