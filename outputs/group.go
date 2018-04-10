// Copyright 2018 Google Inc.
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

package outputs

import (
	"math"

	"github.com/soumya92/barista/bar"
)

// SegmentGroup represents a group of Segments to be
// displayed together on the bar.
type SegmentGroup struct {
	*groupData
}

// We should support both chaining (e.g. Group(...).MinWidth(10).Append(...))
// and sequential calls (e.g. g.InnerSeparators(true); g.Append(...);).
// To do so, SegmentGroup needs to be mutable in-place, but making it
// a reference type will disallow `return Group(...).Padding(...)'.
// To work around this, we wrap a reference type that holds all the data,
// and have each method act on the inner field.

// groupData stores the data required by a SegmentGroup.
type groupData struct {
	segments []bar.Segment
	// To support addition of segments after construction, store
	// attributes on the group, and apply them in Segments().
	attrSet        int
	color          bar.Color
	background     bar.Color
	border         bar.Color
	minWidth       int
	align          bar.TextAlignment
	urgent         bool
	innerSeparator bool
	innerPadding   int
	outerSeparator bool
	outerPadding   int
}

const (
	sgaUrgent int = 1 << iota
	sgaMinWidth
	sgaInnerSeparators
	sgaInnerPadding
	sgaOuterSeparator
	sgaOuterPadding
)

// newSegmentGroup constructs an empty SegmentGroup
func newSegmentGroup() SegmentGroup {
	return SegmentGroup{&groupData{}}
}

// Color sets the color for all segments in the group.
func (g SegmentGroup) Color(color bar.Color) SegmentGroup {
	g.color = color
	return g
}

// Background sets the background color for all segments in the group.
func (g SegmentGroup) Background(background bar.Color) SegmentGroup {
	g.background = background
	return g
}

// Border sets the border color for all segments in the group.
func (g SegmentGroup) Border(border bar.Color) SegmentGroup {
	g.border = border
	return g
}

// Align sets the text alignment for all segments in the group.
func (g SegmentGroup) Align(align bar.TextAlignment) SegmentGroup {
	g.align = align
	return g
}

// Urgent sets the urgency flag for all segments in the group.
func (g SegmentGroup) Urgent(urgent bool) SegmentGroup {
	g.attrSet |= sgaUrgent
	g.urgent = urgent
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
	g.attrSet |= sgaMinWidth
	g.minWidth = minWidth
	return g
}

// Separator sets the separator visibility of the last segment in the group.
func (g SegmentGroup) Separator(separator bool) SegmentGroup {
	g.attrSet |= sgaOuterSeparator
	g.outerSeparator = separator
	return g
}

// Padding sets the padding of the last segment in the group.
func (g SegmentGroup) Padding(separatorWidth int) SegmentGroup {
	g.attrSet |= sgaOuterPadding
	g.outerPadding = separatorWidth
	return g
}

// InnerSeparators sets the separator visibility between segments of this group.
func (g SegmentGroup) InnerSeparators(separator bool) SegmentGroup {
	g.attrSet |= sgaInnerSeparators
	g.innerSeparator = separator
	return g
}

// InnerPadding sets the padding between segments of this group.
func (g SegmentGroup) InnerPadding(separatorWidth int) SegmentGroup {
	g.attrSet |= sgaInnerPadding
	g.innerPadding = separatorWidth
	return g
}

// Glue is a shortcut to remove the inner separators and padding.
func (g SegmentGroup) Glue() SegmentGroup {
	return g.InnerSeparators(false).InnerPadding(0)
}

// Append adds additional segments to this group.
func (g SegmentGroup) Append(segments ...bar.Segment) SegmentGroup {
	g.segments = append(g.segments, segments...)
	return g
}

// isSet returns true if an attribute was set, discarding its value.
func isSet(_ interface{}, isSet bool) bool {
	return isSet
}

// Segments implements bar.Output for SegmentGroup.
// This method is responsible for computing all attributes so that
// all segments, even those added after attributes were set on the group
// correctly reflect those attributes in the final output.
func (g SegmentGroup) Segments() []bar.Segment {
	segments := make([]bar.Segment, 0)
	remainingWidth := float64(g.minWidth)
	if g.attrSet&sgaMinWidth != 0 {
		remainingWidth -= g.existingMinWidth()
	}
	for idx, s := range g.segments {
		c := s.Clone()
		remainingSegments := len(g.segments) - idx
		if remainingSegments == 1 {
			if !isSet(s.HasSeparator()) && g.attrSet&sgaOuterSeparator != 0 {
				c.Separator(g.outerSeparator)
			}
			if !isSet(s.GetPadding()) && g.attrSet&sgaOuterPadding != 0 {
				c.Padding(g.outerPadding)
			}
		} else {
			if !isSet(s.HasSeparator()) && g.attrSet&sgaInnerSeparators != 0 {
				c.Separator(g.innerSeparator)
			}
			if !isSet(s.GetPadding()) && g.attrSet&sgaInnerPadding != 0 {
				c.Padding(g.innerPadding)
			}
		}
		if !isSet(s.GetMinWidth()) && g.attrSet&sgaMinWidth != 0 {
			myWidth := math.Floor(remainingWidth/float64(remainingSegments) + 0.5)
			if myWidth >= 0 {
				c.MinWidth(int(myWidth))
				remainingWidth = remainingWidth - myWidth
			}
		}
		if !isSet(s.GetColor()) && g.color != "" {
			c.Color(g.color)
		}
		if !isSet(s.GetBackground()) && g.background != "" {
			c.Background(g.background)
		}
		if !isSet(s.GetBorder()) && g.border != "" {
			c.Border(g.border)
		}
		if !isSet(s.GetAlignment()) && g.align != "" {
			c.Align(g.align)
		}
		if !isSet(s.IsUrgent()) && g.attrSet&sgaUrgent != 0 {
			c.Urgent(g.urgent)
		}
		segments = append(segments, c)
	}
	return segments
}

// existingMinWidth sums all integral minimum widths from the segments.
// This allows us to distribute the minWidth amongst the other segments
// while keeping the total min width the same as what was given.
func (g SegmentGroup) existingMinWidth() (result float64) {
	for _, s := range g.segments {
		minWidth, isSet := s.GetMinWidth()
		if !isSet {
			continue
		}
		if minWidthPx, ok := minWidth.(int); ok {
			result += float64(minWidthPx)
		}
	}
	return result
}
