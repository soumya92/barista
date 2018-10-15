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
	"fmt"
	"image/color"
	"strconv"

	"github.com/lucasb-eyer/go-colorful"
)

func (n *Node) setAttr(name, value string) *Node {
	if n.content == "" {
		// Convert a placeholder wrapper to a full span tag.
		n.content = "span"
		n.attributes = map[string]string{}
	}
	if value == "" {
		delete(n.attributes, name)
	} else {
		n.attributes[name] = value
	}
	return n
}

func colorAndAlpha(value color.Color) (color, alpha string) {
	if value == nil {
		return "", ""
	}
	_, _, _, a := value.RGBA()
	if a == 0 {
		return "", "0"
	}
	if a < 0xffff {
		alpha = strconv.Itoa(int(a))
	}
	cful, _ := colorful.MakeColor(value)
	color = cful.Hex()
	return color, alpha
}

// Font sets the font face.
func (n *Node) Font(face string) *Node {
	return n.setAttr("face", face)
}

// Size sets the font size, in points.
func (n *Node) Size(size float64) *Node {
	// Pango size is 1/1024ths of a point.
	return n.setAttr("size", strconv.Itoa(int(size*1024)))
}

// Keyword sizes supported in Pango.
//go:generate ruby kwattrs.rb --name=size XXSmall:xx-small XSmall:x-small Small Medium Large XLarge:x-large XXLarge:xx-large

// Smaller decreases the font size of the contents
// by wrapping them in <small>...</small>
func (n *Node) Smaller() *Node {
	n.children = []*Node{&Node{
		nodeType: ntSizer,
		content:  "small",
		children: n.children,
	}}
	return n
}

// Larger increases the font size of the contents
// by wrapping them in <big>...</big>
func (n *Node) Larger() *Node {
	n.children = []*Node{&Node{
		nodeType: ntSizer,
		content:  "big",
		children: n.children,
	}}
	return n
}

// Font styles supported in Pango.
//go:generate ruby kwattrs.rb --name=style StyleNormal:normal Oblique Italic

// Weight sets the font weight in numeric form.
func (n *Node) Weight(weight int) *Node {
	return n.setAttr("weight", strconv.Itoa(weight))
}

// Keyword weights supported in Pango.
//go:generate ruby kwattrs.rb --name=weight UltraLight Light WeightNormal:normal Bold UltraBold Heavy

// Pango font variants.
//go:generate ruby kwattrs.rb --name=variant VariantNormal:normal SmallCaps

// Pango font stretch keywords.
//go:generate ruby kwattrs.rb --name=stretch UltraCondensed ExtraCondensed Condensed SemiCondensed StretchNormal:normal SemiExpanded Expanded ExtraExpanded UltraExpanded

// Color applies a foreground color and alpha.
func (n *Node) Color(c color.Color) *Node {
	col, alpha := colorAndAlpha(c)
	n.setAttr("alpha", alpha)
	return n.setAttr("color", col)
}

// Alpha applies just a foreground alpha, keeping the default text colour.
func (n *Node) Alpha(alpha float64) *Node {
	return n.setAttr("alpha", fmt.Sprintf("%.0f", 65535.0*alpha))
}

// Background applies a background color and alpha.
func (n *Node) Background(c color.Color) *Node {
	col, alpha := colorAndAlpha(c)
	n.setAttr("background_alpha", alpha)
	return n.setAttr("background", col)
}

// Pango underline keywords.
//go:generate ruby kwattrs.rb --name=underline UnderlineNone:none UnderlineSingle:single UnderlineDouble:double UnderlineLow:low UnderlineError:error

// UnderlineColor applies an underline color.
func (n *Node) UnderlineColor(c color.Color) *Node {
	col, _ := colorAndAlpha(c)
	return n.setAttr("underline_color", col)
}

// Rise sets the font "rise" in pango units.
// Negative for subscript, positive for superscript.
func (n *Node) Rise(rise int) *Node {
	return n.setAttr("rise", strconv.Itoa(rise))
}

// Whether to strike through the text.
//go:generate ruby kwattrs.rb --name=strikethrough Strikethrough:true NoStrikethrough:false

// StrikethroughColor applies a strikethrough color.
func (n *Node) StrikethroughColor(c color.Color) *Node {
	col, _ := colorAndAlpha(c)
	return n.setAttr("strikethrough_color", col)
}

// LetterSpacing sets the letter spacing, in points.
func (n *Node) LetterSpacing(spacing float64) *Node {
	// Pango spacing is 1/1024ths of a point.
	return n.setAttr("letter_spacing", strconv.Itoa(int(spacing*1024)))
}
