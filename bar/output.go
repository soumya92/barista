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

// SegmentGroup represents a group of Segments to be
// displayed together on the bar.
type SegmentGroup []Segment

// Color sets the color for all segments in the group.
func (g SegmentGroup) Color(color Color) SegmentGroup {
	for _, s := range g {
		s.Color(color)
	}
	return g
}

// Background sets the background color for all segments in the group.
func (g SegmentGroup) Background(background Color) SegmentGroup {
	for _, s := range g {
		s.Background(background)
	}
	return g
}

// Border sets the border color for all segments in the group.
func (g SegmentGroup) Border(border Color) SegmentGroup {
	for _, s := range g {
		s.Border(border)
	}
	return g
}

// Align sets the text alignment for all segments in the group.
func (g SegmentGroup) Align(align TextAlignment) SegmentGroup {
	for _, s := range g {
		s.Align(align)
	}
	return g
}

// Urgent sets the urgency flag for all segments in the group.
func (g SegmentGroup) Urgent(urgent bool) SegmentGroup {
	for _, s := range g {
		s.Urgent(urgent)
	}
	return g
}

// Markup sets the markup type for all segments in the group.
func (g SegmentGroup) Markup(markup Markup) SegmentGroup {
	for _, s := range g {
		s.Markup(markup)
	}
	return g
}

/*
Width and separator(width) are treated specially such that the methods
make sense when called on a single-segment output (such as the result
of outputs.Textf(...)) as well as when called on a multi-segment group.

To that end, min-width distributes the minimum width equally amongst
all segments, and separator(width) only operate on the last segment.

Additional methods for "inner" separator(width) operate on all but the
last segment.
*/

// MinWidth sets the minimum width for the output, by (mostly) equally
// distributing the given minWidth amongst all segments in the group.
func (g SegmentGroup) MinWidth(minWidth int) SegmentGroup {
	remainingWidth := float64(minWidth)
	for idx, s := range g {
		remainingSegments := float64(len(g) - idx)
		myWidth := math.Floor(remainingWidth/remainingSegments + 0.5)
		s.MinWidth(int(myWidth))
		remainingWidth = remainingWidth - myWidth
	}
	return g
}

// Separator sets the separator visibility of the last segment in the group.
func (g SegmentGroup) Separator(separator bool) SegmentGroup {
	if len(g) > 0 {
		g[len(g)-1].Separator(separator)
	}
	return g
}

// SeparatorWidth sets the separator width of the last segment in the group.
func (g SegmentGroup) SeparatorWidth(separatorWidth int) SegmentGroup {
	if len(g) > 0 {
		g[len(g)-1].SeparatorWidth(separatorWidth)
	}
	return g
}

// InnerSeparator sets the separator visibility between segments of this group.
func (g SegmentGroup) InnerSeparator(separator bool) SegmentGroup {
	for idx, s := range g {
		if idx+1 < len(g) {
			s.Separator(separator)
		}
	}
	return g
}

// InnerSeparatorWidth sets the separator width between segments of this group.
func (g SegmentGroup) InnerSeparatorWidth(separatorWidth int) SegmentGroup {
	for idx, s := range g {
		if idx+1 < len(g) {
			s.SeparatorWidth(separatorWidth)
		}
	}
	return g
}

// Segments trivially implements bar.Output for SegmentGroup.
func (g SegmentGroup) Segments() []Segment {
	return g
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

// Segments implements bar.Output for a single Segment.
func (s Segment) Segments() []Segment {
	return []Segment{s}
}
