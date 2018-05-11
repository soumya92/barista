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

	"github.com/lucasb-eyer/go-colorful"
)

// Font sets the font face.
func Font(face string) Attribute {
	return Attribute{"face", face}
}

// Size sets the font size, in points.
func Size(size float64) Attribute {
	// Pango size is 1/1024ths of a point.
	return Attribute{"size", fmt.Sprintf("%d", int(size*1024))}
}

// Keyword sizes supported in Pango.
var (
	XXSmall = Attribute{"size", "xx-small"}
	XSmall  = Attribute{"size", "x-small"}
	Small   = Attribute{"size", "small"}
	Medium  = Attribute{"size", "medium"}
	Large   = Attribute{"size", "large"}
	XLarge  = Attribute{"size", "x-large"}
	XXLarge = Attribute{"size", "xx-large"}

	Smaller = Attribute{"size", "smaller"}
	Larger  = Attribute{"size", "larger"}
)

// Font styles supported in Pango.
var (
	StyleNormal = Attribute{"style", "normal"}
	Oblique     = Attribute{"style", "oblique"}
	Italic      = Attribute{"style", "italic"}
)

// Weight sets the font weight in numeric form.
func Weight(weight int) Attribute {
	return Attribute{"weight", fmt.Sprintf("%d", weight)}
}

// Keyword weights supported in Pango.
var (
	Ultralight   = Attribute{"weight", "ultralight"}
	Light        = Attribute{"weight", "light"}
	WeightNormal = Attribute{"weight", "normal"}
	Bold         = Attribute{"weight", "bold"}
	UltraBold    = Attribute{"weight", "ultrabold"}
	Heavy        = Attribute{"weight", "heavy"}
)

// Pango font variants.
var (
	VariantNormal = Attribute{"variant", "normal"}
	SmallCaps     = Attribute{"variant", "smallcaps"}
)

// Pango font stretch keywords.
var (
	StretchNormal  = Attribute{"stretch", "normal"}
	UltraCondensed = Attribute{"stretch", "ultracondensed"}
	ExtraCondensed = Attribute{"stretch", "extracondensed"}
	Condensed      = Attribute{"stretch", "condensed"}
	SemiCondensed  = Attribute{"stretch", "semicondensed"}
	SemiExpanded   = Attribute{"stretch", "semiexpanded"}
	Expanded       = Attribute{"stretch", "expanded"}
	ExtraExpanded  = Attribute{"stretch", "extraexpanded"}
	UltraExpanded  = Attribute{"stretch", "ultraexpanded"}
)

func colorAttrs(name, alpha string, value color.Color) (attrs []Attribute) {
	_, _, _, a := value.RGBA()
	if a == 0 {
		if alpha != "" {
			attrs = append(attrs, Attribute{alpha, "0"})
		}
		return // attrs
	}
	if a < 0xffff && alpha != "" {
		attrs = append(attrs, Attribute{alpha, fmt.Sprintf("%d", a)})
	}
	attrs = append(attrs, Attribute{name, colorful.MakeColor(value).Hex()})
	return // attrs
}

// Color applies a foreground color and alpha.
func Color(c color.Color) []Attribute {
	return colorAttrs("color", "alpha", c)
}

// Background applies a background color and alpha.
func Background(c color.Color) []Attribute {
	return colorAttrs("background", "background_alpha", c)
}

// Pango underline keywords.
var (
	UnderlineNone   = Attribute{"underline", "none"}
	UnderlineSingle = Attribute{"underline", "single"}
	UnderlineDouble = Attribute{"underline", "double"}
	UnderlineLow    = Attribute{"underline", "low"}
	UnderlineError  = Attribute{"underline", "error"}
)

// UnderlineColor applies an underline color.
func UnderlineColor(c color.Color) []Attribute {
	return colorAttrs("underline_color", "", c)
}

// Rise sets the font "rise" in pango units.
// Negative for subscript, positive for superscript.
func Rise(rise int) Attribute {
	return Attribute{"rise", fmt.Sprintf("%d", rise)}
}

// Whether to strike through the text.
var (
	Strikethrough   = Attribute{"strikethrough", "true"}
	NoStrikethrough = Attribute{"strikethrough", "false"}
)

// StrikethroughColor applies a strikethrough color.
func StrikethroughColor(c color.Color) []Attribute {
	return colorAttrs("strikethrough_color", "", c)
}

// LetterSpacing sets the letter spacing, in points.
func LetterSpacing(spacing float64) Attribute {
	// Pango spacing is 1/1024ths of a point.
	return Attribute{"letter_spacing", fmt.Sprintf("%d", int(spacing*1024))}
}
