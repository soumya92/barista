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

	"github.com/soumya92/barista/bar"
)

// Font sets the font face.
type Font string

// PangoAttr returns the font as a pango 'face' value.
func (f Font) PangoAttr() (string, string) {
	return "face", string(f)
}

// Size sets the font size, in points.
type Size float64

// PangoAttr returns the font size as a pango 'size' value.
func (s Size) PangoAttr() (string, string) {
	// Pango size is 1/1024ths of a point.
	return "size", fmt.Sprintf("%d", int(float64(s)*1024))
}

type size string

// Keyword sizes supported in Pango.
var (
	XXSmall Attribute = size("xx-small")
	XSmall            = size("x-small")
	Small             = size("small")
	Medium            = size("medium")
	Large             = size("large")
	XLarge            = size("x-large")
	XXLarge           = size("xx-large")

	Smaller = size("smaller")
	Larger  = size("larger")
)

// PangoAttr returns the font size as a pango 'size' keyword value.
func (s size) PangoAttr() (string, string) {
	return "size", string(s)
}

type style string

// Font styles supported in Pango.
var (
	StyleNormal Attribute = style("normal")
	Oblique               = style("oblique")
	Italic                = style("italic")
)

// PangoAttr returns the font as a pango 'style' value.
func (s style) PangoAttr() (string, string) {
	return "style", string(s)
}

// Weight sets the font weight in numeric form.
type Weight int

// PangoAttr returns the weight as a pango 'weight' value.
func (w Weight) PangoAttr() (string, string) {
	return "weight", fmt.Sprintf("%d", w)
}

type weight string

// Keyword weights supported in Pango.
var (
	Ultralight   Attribute = weight("ultralight")
	Light                  = weight("light")
	WeightNormal           = weight("normal")
	Bold                   = weight("bold")
	UltraBold              = weight("ultrabold")
	Heavy                  = weight("heavy")
)

// PangoAttr returns the weight as a pango 'weight' value.
func (w weight) PangoAttr() (string, string) {
	return "weight", string(w)
}

type variant string

// Pango font variants.
var (
	VariantNormal Attribute = variant("normal")
	SmallCaps               = variant("smallcaps")
)

// PangoAttr returns the variant as a pango 'variant' value.
func (v variant) PangoAttr() (string, string) {
	return "variant", string(v)
}

type stretch string

// Pango font stretch keywords.
var (
	StretchNormal  Attribute = stretch("normal")
	UltraCondensed           = stretch("ultracondensed")
	ExtraCondensed           = stretch("extracondensed")
	Condensed                = stretch("condensed")
	SemiCondensed            = stretch("semicondensed")
	SemiExpanded             = stretch("semiexpanded")
	Expanded                 = stretch("expanded")
	ExtraExpanded            = stretch("extraexpanded")
	UltraExpanded            = stretch("ultraexpanded")
)

// PangoAttr returns the stretch as a pango 'stretch' value.
func (s stretch) PangoAttr() (string, string) {
	return "stretch", string(s)
}

// Background wraps a bar color but applies it as a background
// instead of the foreground.
type Background bar.Color

// PangoAttr delegates to bar.Color to return the pango color value.
func (b Background) PangoAttr() (string, string) {
	return "background", bar.Color(b).String()
}

// Alpha sets the foreground opacity on a scale of 0 to 1.
type Alpha float64

// PangoAttr returns the fg alpha as a pango 'alpha' value.
func (a Alpha) PangoAttr() (string, string) {
	// Pango alpha ranges from 1 to 65535.
	return "alpha", fmt.Sprintf("%d", int(float64(a)*65535))
}

// BgAlpha sets the background opacity on a scale of 0 to 1.
type BgAlpha float64

// PangoAttr returns the bg alpha as a pango 'background_alpha' value.
func (b BgAlpha) PangoAttr() (string, string) {
	// Pango alpha ranges from 1 to 65535.
	return "background_alpha", fmt.Sprintf("%d", int(float64(b)*65535))
}

type underline string

// Pango underline keywords.
var (
	UnderlineNone   Attribute = underline("none")
	UnderlineSingle           = underline("single")
	UnderlineDouble           = underline("double")
	UnderlineLow              = underline("low")
	UnderlineError            = underline("error")
)

// PangoAttr returns the underline as a pango 'underline' value.
func (u underline) PangoAttr() (string, string) {
	return "underline", string(u)
}

// UnderlineColor wraps a bar color but applies it as the
// underline color instead of the foreground.
type UnderlineColor bar.Color

// PangoAttr delegates to bar.Color to return the pango color value.
func (u UnderlineColor) PangoAttr() (string, string) {
	return "underline_color", bar.Color(u).String()
}

// Rise sets the font "rise" in pango units.
// Negative for subscript, positive for superscript.
type Rise int

// PangoAttr returns the rise as a pango 'rise' value.
func (r Rise) PangoAttr() (string, string) {
	return "rise", fmt.Sprintf("%d", r)
}

type strikethrough bool

// Whether to strike through the text.
var (
	Strikethrough   Attribute = strikethrough(true)
	NoStrikethrough           = strikethrough(false)
)

// PangoAttr returns true or false for the pango 'strikethrough' attribute.
func (s strikethrough) PangoAttr() (name string, value string) {
	name = "strikethrough"
	if s {
		value = "true"
	} else {
		value = "false"
	}
	return name, value
}

// StrikethroughColor wraps a bar color but applies it as the
// strikethrough color instead of the foreground.
type StrikethroughColor bar.Color

// PangoAttr delegates to bar.Color to return the pango color value.
func (s StrikethroughColor) PangoAttr() (string, string) {
	return "strikethrough_color", bar.Color(s).String()
}

// LetterSpacing sets the letter spacing, in points.
type LetterSpacing float64

// PangoAttr returns the letter spacing as a pango 'letter_spacing' value.
func (l LetterSpacing) PangoAttr() (string, string) {
	// Pango spacing is 1/1024ths of a point.
	return "letter_spacing", fmt.Sprintf("%d", int(float64(l)*1024))
}
