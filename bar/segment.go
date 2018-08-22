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

import "image/color"

// TextSegment creates a new output segment with text content.
func TextSegment(text string) *Segment {
	return new(Segment).Text(text)
}

// PangoSegment creates a new output segment with content that uses pango
// markup for formatting. Not all features may be supported.
// See https://developer.gnome.org/pango/stable/PangoMarkupFormat.html.
func PangoSegment(text string) *Segment {
	return new(Segment).Pango(text)
}

// ErrorSegment creates a new output segment that displays an error.
// On the bar itself, it's an urgent segment showing 'Error' or '!'
// based on available space, but the full error will be shown using
// i3-nagbar when the segment is right-clicked.
func ErrorSegment(e error) *Segment {
	return TextSegment("Error").Error(e).ShortText("!").Urgent(true)
}

// Text sets the text content of this segment. It clears any previous
// content and resets the markup style.
func (s *Segment) Text(content string) *Segment {
	s.text = content
	s.pango = false
	return s
}

// Pango sets the pango content of this segment. It clears any previous
// content and sets the markup style to pango.
func (s *Segment) Pango(content string) *Segment {
	s.text = content
	s.pango = true
	return s
}

// Content returns the text content of the segment, and whether or not
// it is using pango markup.
func (s *Segment) Content() (text string, isPango bool) {
	return s.text, s.pango
}

// ShortText sets the shortened text, used if the default text
// for all segments does not fit in the bar.
func (s *Segment) ShortText(shortText string) *Segment {
	s.shortText = shortText
	s.attrSet |= saShortText
	return s
}

// GetShortText returns the short text of this segment.
// The second value indicates whether it was explicitly set.
func (s *Segment) GetShortText() (string, bool) {
	return s.shortText, s.attrSet&saShortText != 0
}

// Error associates an error with the segment. Setting an error
// changes event handling to display the full error text on left
// click, and restart the module on right/middle click.
// (If the module is still running, right/middle click is a no-op).
func (s *Segment) Error(err error) *Segment {
	s.err = err
	return s
}

// GetError returns any error associated with this segment
// or nil if no error is associated with this segment.
func (s *Segment) GetError() error {
	return s.err
}

// Color sets the foreground color for the segment.
func (s *Segment) Color(color color.Color) *Segment {
	s.color = color
	return s
}

// GetColor returns the foreground color of this segment.
// The second value indicates whether it was explicitly set.
func (s *Segment) GetColor() (color.Color, bool) {
	return s.color, s.color != nil
}

// Background sets the background color for the segment.
func (s *Segment) Background(background color.Color) *Segment {
	s.background = background
	return s
}

// GetBackground returns the background color of this segment.
// The second value indicates whether it was explicitly set.
func (s *Segment) GetBackground() (color.Color, bool) {
	return s.background, s.background != nil
}

// Border sets the border color for the segment.
func (s *Segment) Border(border color.Color) *Segment {
	s.border = border
	return s
}

// GetBorder returns the border color of this segment.
// The second value indicates whether it was explicitly set.
func (s *Segment) GetBorder() (color.Color, bool) {
	return s.border, s.border != nil
}

// MinWidth sets the minimum width for the segment.
func (s *Segment) MinWidth(minWidth int) *Segment {
	s.minWidth = minWidth
	return s
}

// MinWidthPlaceholder sets the minimum width of the segment such that
// the placeholder string will fit.
func (s *Segment) MinWidthPlaceholder(placeholder string) *Segment {
	s.minWidth = placeholder
	return s
}

// GetMinWidth returns the minimum width of this segment.
// The returned value will either be an int or a string, based
// on how it was originally set.
// The second value indicates whether it was explicitly set.
func (s *Segment) GetMinWidth() (interface{}, bool) {
	return s.minWidth, s.minWidth != nil
}

// Align sets the text alignment within the segment.
func (s *Segment) Align(align TextAlignment) *Segment {
	s.align = align
	return s
}

// GetAlignment returns the text alignment of this segment.
// The second value indicates whether it was explicitly set.
func (s *Segment) GetAlignment() (TextAlignment, bool) {
	return s.align, s.align != ""
}

// Urgent sets the urgency of the segment.
func (s *Segment) Urgent(urgent bool) *Segment {
	s.urgent = urgent
	s.attrSet |= saUrgent
	return s
}

// IsUrgent returns true if this segment is marked urgent.
// The second value indicates whether it was explicitly set.
func (s *Segment) IsUrgent() (bool, bool) {
	return s.urgent, s.attrSet&saUrgent != 0
}

// Separator controls whether this *Segment has a separator.
func (s *Segment) Separator(separator bool) *Segment {
	s.separator = separator
	s.attrSet |= saSeparator
	return s
}

// HasSeparator returns true if the segment has a separator.
// The second value indicates whether it was explicitly set.
func (s *Segment) HasSeparator() (bool, bool) {
	if s.attrSet&saSeparator != 0 {
		return s.separator, true
	}
	// Default value for separator is true in i3.
	return true, false
}

// Padding sets the padding at the end of this segment. The separator
// (if displayed) will be centred within the padding.
func (s *Segment) Padding(padding int) *Segment {
	s.padding = padding
	s.attrSet |= saPadding
	return s
}

// GetPadding returns the padding at the end of this segment.
// The second value indicates whether it was explicitly set.
// This maps to "separator_block_width" in i3.
func (s *Segment) GetPadding() (int, bool) {
	if s.attrSet&saPadding != 0 {
		return s.padding, true
	}
	// Default padding is 9px.
	return 9, false
}

// OnClick sets a function to be called when the segment is clicked.
// A nil function is treated as equivalent to func(Event) {}, which
// means CanClick() will return true, but Click(Event) will do nothing.
// Nil can therefore be used to prevent module-level default handlers
// from being attached to a segment.
func (s *Segment) OnClick(fn func(Event)) *Segment {
	if fn == nil {
		fn = func(Event) {}
	}
	s.onClick = fn
	return s
}

// HasClick returns whether this segment has a click handler defined.
// Modules can use this check to assign default handlers to segments
// where the user has not already assigned a click handler.
func (s *Segment) HasClick() bool {
	return s.onClick != nil
}

// Click calls a previously set click handler with the given Event.
func (s *Segment) Click(e Event) {
	if s.onClick != nil {
		s.onClick(e)
	}
}

// Segments implements bar.Output for a single Segment.
func (s *Segment) Segments() []*Segment {
	return []*Segment{s}
}

// Clone makes a copy of the Segment that can be modified
// without the changes being reflected in the original.
func (s *Segment) Clone() *Segment {
	copied := &Segment{}
	*copied = *s
	return copied
}

// Segments returns the list of segments as a bar.Output.
func (s Segments) Segments() []*Segment {
	return s
}
