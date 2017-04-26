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

// AttrName returns the name of the pango 'face' attribute.
func (f Font) AttrName() string {
	return "face"
}

// AttrValue returns the font as a pango 'face' value.
func (f Font) AttrValue() string {
	return string(f)
}

// Size sets the font size, in points.
type Size float64

// AttrName returns the name of the pango 'size' attribute.
func (s Size) AttrName() string {
	return "size"
}

// AttrValue returns the font size as a pango 'size' value.
func (s Size) AttrValue() string {
	// Pango size is 1/1024ths of a point.
	return fmt.Sprintf("%d", int(float64(s)*1024))
}

type size string

var (
	// Keyword sizes supported in Pango.
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

// AttrName returns the name of the pango 'size' attribute.
func (s size) AttrName() string {
	return "size"
}

// AttrValue returns the font size as a pango 'size' keyword value.
func (s size) AttrValue() string {
	return string(s)
}

type style string

var (
	// Font styles supported in Pango.
	StyleNormal Attribute = style("normal")
	Oblique               = style("oblique")
	Italic                = style("italic")
)

// AttrName returns the name of the pango 'style' attribute.
func (s style) AttrName() string {
	return "style"
}

// AttrValue returns the font as a pango 'style' value.
func (s style) AttrValue() string {
	return string(s)
}

// Weight sets the font weight in numeric form.
type Weight int

// AttrName returns the name of the pango 'weight' attribute.
func (w Weight) AttrName() string {
	return "weight"
}

// AttrValue returns the weight as a pango 'weight' value.
func (w Weight) AttrValue() string {
	return fmt.Sprintf("%d", w)
}

type weight string

var (
	// Keyword weights supported in Pango.
	Ultralight   Attribute = weight("ultralight")
	Light                  = weight("light")
	WeightNormal           = weight("normal")
	Bold                   = weight("bold")
	UltraBold              = weight("ultrabold")
	Heavy                  = weight("heavy")
)

// AttrName returns the name of the pango 'weight' attribute.
func (w weight) AttrName() string {
	return "weight"
}

// AttrValue returns the weight as a pango 'weight' value.
func (w weight) AttrValue() string {
	return string(w)
}

type variant string

var (
	// Pango font variants.
	VariantNormal Attribute = variant("normal")
	SmallCaps               = variant("smallcaps")
)

// AttrName returns the name of the pango 'variant' attribute.
func (v variant) AttrName() string {
	return "variant"
}

// AttrValue returns the variant as a pango 'variant' value.
func (v variant) AttrValue() string {
	return string(v)
}

type stretch string

var (
	// Pango font stretch keywords.
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

// AttrName returns the name of the pango 'stretch' attribute.
func (s stretch) AttrName() string {
	return "stretch"
}

// AttrValue returns the stretch as a pango 'stretch' value.
func (s stretch) AttrValue() string {
	return string(s)
}

// Background wraps a bar color but applies it as a background
// instead of the foreground.
type Background bar.Color

// AttrName returns the name of the pango 'background' attribute.
func (b Background) AttrName() string {
	return "background"
}

// Alpha sets the foreground opacity on a scale of 0 to 1.
type Alpha float64

// AttrName returns the name of the pango 'alpha' attribute.
func (a Alpha) AttrName() string {
	return "alpha"
}

// AttrValue returns the fg alpha as a pango 'alpha' value.
func (a Alpha) AttrValue() string {
	// Pango alpha ranges from 1 to 65535.
	return fmt.Sprintf("%d", int(float64(a)*65535))
}

// BgAlpha sets the background opacity on a scale of 0 to 1.
type BgAlpha float64

// AttrName returns the name of the pango 'background_alpha' attribute.
func (b BgAlpha) AttrName() string {
	return "background_alpha"
}

// AttrValue returns the bg alpha as a pango 'background_alpha' value.
func (b BgAlpha) AttrValue() string {
	// Pango alpha ranges from 1 to 65535.
	return fmt.Sprintf("%d", int(float64(b)*65535))
}

type underline string

var (
	// Pango underline keywords.
	UnderlineNone   Attribute = underline("none")
	UnderlineSingle           = underline("single")
	UnderlineDouble           = underline("double")
	UnderlineLow              = underline("low")
	UnderlineError            = underline("error")
)

// AttrName returns the name of the pango 'underline' attribute.
func (u underline) AttrName() string {
	return "underline"
}

// AttrValue returns the underline as a pango 'underline' value.
func (u underline) AttrValue() string {
	return string(u)
}

// UnderlineColor wraps a bar color but applies it as the
// underline color instead of the foreground.
type UnderlineColor bar.Color

// AttrName returns the name of the pango 'underline_color' attribute.
func (u UnderlineColor) AttrName() string {
	return "underline_color"
}

// Rise sets the font "rise" in pango units.
// Negative for subscript, positive for superscript.
type Rise int

// AttrName returns the name of the pango 'rise' attribute.
func (r Rise) AttrName() string {
	return "rise"
}

// AttrValue returns the rise as a pango 'rise' value.
func (r Rise) AttrValue() string {
	return fmt.Sprintf("%d", r)
}

type strikethrough bool

var (
	// Whether to strike through the text.
	Strikethrough   Attribute = strikethrough(true)
	NoStrikethrough           = strikethrough(false)
)

// AttrName returns the name of the pango 'strikethrough' attribute.
func (s strikethrough) AttrName() string {
	return "strikethrough"
}

// AttrValue returns true or false for the pango 'strikethrough' attribute.
func (s strikethrough) AttrValue() string {
	return fmt.Sprintf("%v", s)
}

// StrikethroughColor wraps a bar color but applies it as the
// strikethrough color instead of the foreground.
type StrikethroughColor bar.Color

// AttrName returns the name of the pango 'strikethrough_color' attribute.
func (s StrikethroughColor) AttrName() string {
	return "strikethrough_color"
}

// LetterSpacing sets the letter spacing, in points.
type LetterSpacing float64

// AttrName returns the name of the pango 'letter_spacing' attribute.
func (l LetterSpacing) AttrName() string {
	return "letter_spacing"
}

// AttrValue returns the letter spacing as a pango 'letter_spacing' value.
func (l LetterSpacing) AttrValue() string {
	// Pango spacing is 1/1024ths of a point.
	return fmt.Sprintf("%d", int(float64(l)*1024))
}
