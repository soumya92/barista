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

// TextSegment creates a new output segment with text content.
func TextSegment(text string) Segment {
	return Segment{&data{text: text, markup: "none"}}
}

// PangoSegment creates a new output segment with content that uses pango
// markup for formatting. Not all features may be supported.
// See https://developer.gnome.org/pango/stable/PangoMarkupFormat.html.
func PangoSegment(text string) Segment {
	return Segment{&data{text: text, markup: "pango"}}
}

// Text returns the text content of this segment.
func (s Segment) Text() string {
	return s.text
}

// IsPango returns true if the segment is using pango markup.
func (s Segment) IsPango() bool {
	return s.markup == "pango"
}

// ShortText sets the shortened text, used if the default text
// for all segments does not fit in the bar.
func (s Segment) ShortText(shortText string) Segment {
	s.shortText = shortText
	s.attrSet |= saShortText
	return s
}

// GetShortText returns the short text of this segment.
// The second value indicates whether it was explicitly set.
func (s Segment) GetShortText() (string, bool) {
	return s.shortText, s.attrSet&saShortText != 0
}

// Color sets the foreground color for the segment.
func (s Segment) Color(color Color) Segment {
	s.color = color
	return s
}

// GetColor returns the foreground color of this segment.
// The second value indicates whether it was explicitly set.
func (s Segment) GetColor() (Color, bool) {
	return s.color, s.color != ""
}

// Background sets the background color for the segment.
func (s Segment) Background(background Color) Segment {
	s.background = background
	return s
}

// GetBackground returns the background color of this segment.
// The second value indicates whether it was explicitly set.
func (s Segment) GetBackground() (Color, bool) {
	return s.background, s.background != ""
}

// Border sets the border color for the segment.
func (s Segment) Border(border Color) Segment {
	s.border = border
	return s
}

// GetBorder returns the border color of this segment.
// The second value indicates whether it was explicitly set.
func (s Segment) GetBorder() (Color, bool) {
	return s.border, s.border != ""
}

// MinWidth sets the minimum width for the segment.
func (s Segment) MinWidth(minWidth int) Segment {
	s.minWidth = minWidth
	return s
}

// MinWidthPlaceholder sets the minimum width of the segment such that
// the placeholder string will fit.
func (s Segment) MinWidthPlaceholder(placeholder string) Segment {
	s.minWidth = placeholder
	return s
}

// GetMinWidth returns the minimum width of this segment.
// The returned value will either be an int or a string, based
// on how it was originally set.
// The second value indicates whether it was explicitly set.
func (s Segment) GetMinWidth() (interface{}, bool) {
	return s.minWidth, s.minWidth != nil
}

// Align sets the text alignment within the segment.
func (s Segment) Align(align TextAlignment) Segment {
	s.align = align
	return s
}

// GetAlignment returns the text alignment of this segment.
// The second value indicates whether it was explicitly set.
func (s Segment) GetAlignment() (TextAlignment, bool) {
	return s.align, s.align != ""
}

// Urgent sets the urgency of the segment.
func (s Segment) Urgent(urgent bool) Segment {
	s.urgent = urgent
	s.attrSet |= saUrgent
	return s
}

// IsUrgent returns true if this segment is marked urgent.
// The second value indicates whether it was explicitly set.
func (s Segment) IsUrgent() (bool, bool) {
	return s.urgent, s.attrSet&saUrgent != 0
}

// Separator controls whether this Segment has a separator.
func (s Segment) Separator(separator bool) Segment {
	s.separator = separator
	s.attrSet |= saSeparator
	return s
}

// HasSeparator returns true if the segment has a separator.
// The second value indicates whether it was explicitly set.
func (s Segment) HasSeparator() (bool, bool) {
	if s.attrSet&saSeparator != 0 {
		return s.separator, true
	}
	// Default value for separator is true in i3.
	return true, false
}

// Padding sets the padding at the end of this segment. The separator
// (if displayed) will be centred within the padding.
func (s Segment) Padding(padding int) Segment {
	s.padding = padding
	s.attrSet |= saPadding
	return s
}

// GetPadding returns the padding at the end of this segment.
// The second value indicates whether it was explicitly set.
// This maps to "separator_block_width" in i3.
func (s Segment) GetPadding() (int, bool) {
	if s.attrSet&saPadding != 0 {
		return s.padding, true
	}
	// Default padding is 9px.
	return 9, false
}

// Identifier sets an opaque identifier for this Segment.
// This identifier will be passed unchanged to i3bar, and will be used
// as the value of SegmentID in click events originating on this segment.
func (s Segment) Identifier(identifier string) Segment {
	s.identifier = identifier
	return s
}

// GetID returns the identifier for this segment.
// The second value indicates whether it was explicitly set.
func (s Segment) GetID() (string, bool) {
	return s.identifier, s.identifier != ""
}

// Segments implements bar.Output for a single Segment.
func (s Segment) Segments() []Segment {
	return []Segment{s}
}

// Clone makes a copy of the Segment that can be modified
// without the changes being reflected in the original.
func (s Segment) Clone() Segment {
	copied := Segment{&data{}}
	*copied.data = *s.data
	return copied
}
