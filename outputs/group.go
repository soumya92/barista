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
	"image/color"
	"math"
	"time"

	"barista.run/bar"
	"barista.run/timing"
)

// SegmentGroup represents a group of Segments to be
// displayed together on the bar.
type SegmentGroup struct {
	outputs      []bar.Output
	timedOutputs []bar.TimedOutput
	startTime    time.Time
	// To support addition of segments after construction, store
	// attributes on the group, and apply them in Segments().
	attrSet        int
	clickHandler   func(bar.Event)
	color          color.Color
	background     color.Color
	border         color.Color
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

// OnClick sets the default click handler for the group. Any segments
// that don't already have a click handler will delegate to this one.
func (g *SegmentGroup) OnClick(f func(bar.Event)) *SegmentGroup {
	g.clickHandler = f
	return g
}

// Color sets the color for all segments in the group.
func (g *SegmentGroup) Color(color color.Color) *SegmentGroup {
	g.color = color
	return g
}

// Background sets the background color for all segments in the group.
func (g *SegmentGroup) Background(background color.Color) *SegmentGroup {
	g.background = background
	return g
}

// Border sets the border color for all segments in the group.
func (g *SegmentGroup) Border(border color.Color) *SegmentGroup {
	g.border = border
	return g
}

// Align sets the text alignment for all segments in the group.
func (g *SegmentGroup) Align(align bar.TextAlignment) *SegmentGroup {
	g.align = align
	return g
}

// Urgent sets the urgency flag for all segments in the group.
func (g *SegmentGroup) Urgent(urgent bool) *SegmentGroup {
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
func (g *SegmentGroup) MinWidth(minWidth int) *SegmentGroup {
	g.attrSet |= sgaMinWidth
	g.minWidth = minWidth
	return g
}

// Separator sets the separator visibility of the last segment in the group.
func (g *SegmentGroup) Separator(separator bool) *SegmentGroup {
	g.attrSet |= sgaOuterSeparator
	g.outerSeparator = separator
	return g
}

// Padding sets the padding of the last segment in the group.
func (g *SegmentGroup) Padding(separatorWidth int) *SegmentGroup {
	g.attrSet |= sgaOuterPadding
	g.outerPadding = separatorWidth
	return g
}

// InnerSeparators sets the separator visibility between segments of this group.
func (g *SegmentGroup) InnerSeparators(separator bool) *SegmentGroup {
	g.attrSet |= sgaInnerSeparators
	g.innerSeparator = separator
	return g
}

// InnerPadding sets the padding between segments of this group.
func (g *SegmentGroup) InnerPadding(separatorWidth int) *SegmentGroup {
	g.attrSet |= sgaInnerPadding
	g.innerPadding = separatorWidth
	return g
}

// Glue is a shortcut to remove the inner separators and padding.
func (g *SegmentGroup) Glue() *SegmentGroup {
	return g.InnerSeparators(false).InnerPadding(0)
}

// Append adds an additional output to this group.
func (g *SegmentGroup) Append(output bar.Output) *SegmentGroup {
	if g.startTime.IsZero() {
		g.startTime = timing.Now()
	}
	if output == nil {
		return g
	}
	output = resetStartTime(output, g.startTime)
	g.outputs = append(g.outputs, output)
	if to, ok := output.(bar.TimedOutput); ok {
		g.timedOutputs = append(g.timedOutputs, to)
	}
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
func (g *SegmentGroup) Segments() []*bar.Segment {
	var segments []*bar.Segment
	for _, o := range g.outputs {
		for _, s := range o.Segments() {
			segments = append(segments, s.Clone())
		}
	}
	remainingWidth := float64(g.minWidth)
	if g.attrSet&sgaMinWidth != 0 {
		remainingWidth -= existingMinWidth(segments)
	}
	for idx, s := range segments {
		remainingSegments := len(segments) - idx
		if remainingSegments == 1 {
			if !isSet(s.HasSeparator()) && g.attrSet&sgaOuterSeparator != 0 {
				s.Separator(g.outerSeparator)
			}
			if !isSet(s.GetPadding()) && g.attrSet&sgaOuterPadding != 0 {
				s.Padding(g.outerPadding)
			}
		} else {
			if !isSet(s.HasSeparator()) && g.attrSet&sgaInnerSeparators != 0 {
				s.Separator(g.innerSeparator)
			}
			if !isSet(s.GetPadding()) && g.attrSet&sgaInnerPadding != 0 {
				s.Padding(g.innerPadding)
			}
		}
		if !isSet(s.GetMinWidth()) && g.attrSet&sgaMinWidth != 0 {
			myWidth := math.Floor(remainingWidth/float64(remainingSegments) + 0.5)
			if myWidth >= 0 {
				s.MinWidth(int(myWidth))
				remainingWidth = remainingWidth - myWidth
			}
		}
		if !isSet(s.GetColor()) && g.color != nil {
			s.Color(g.color)
		}
		if !isSet(s.GetBackground()) && g.background != nil {
			s.Background(g.background)
		}
		if !isSet(s.GetBorder()) && g.border != nil {
			s.Border(g.border)
		}
		if !isSet(s.GetAlignment()) && g.align != "" {
			s.Align(g.align)
		}
		if !isSet(s.IsUrgent()) && g.attrSet&sgaUrgent != 0 {
			s.Urgent(g.urgent)
		}
		if !s.HasClick() && g.clickHandler != nil {
			s.OnClick(g.clickHandler)
		}
	}
	return segments
}

// NextRefresh handles the case of one or more TimedOutputs being added to the
// group. If any timed outputs exist, NextRefresh will return the earliest next
// refresh time of any of them.
func (g *SegmentGroup) NextRefresh() time.Time {
	var refresh time.Time
	for _, o := range g.timedOutputs {
		next := o.NextRefresh()
		if next.IsZero() {
			continue
		}
		if refresh.IsZero() || next.Before(refresh) {
			refresh = next
		}
	}
	return refresh
}

// existingMinWidth sums all integral minimum widths from the segments.
// This allows us to distribute the minWidth amongst the other segments
// while keeping the total min width the same as what was given.
func existingMinWidth(segments []*bar.Segment) (result float64) {
	for _, s := range segments {
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
