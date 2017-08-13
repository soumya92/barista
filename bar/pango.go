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

// Makes bar.Color conform to the pango Attribute specification.
// This enables code like:
//  pango.Span(
//    "bad-thing",
//    colors.Scheme("bad"),
//  )
// to produce: <span color="red">bad-thing</span>
// assuming that the current scheme's 'bad' color is 'red'.

// AttrName returns the name of the pango 'color' attribute.
func (c Color) AttrName() string {
	return "color"
}

// AttrValue returns the color as a pango 'color' value.
func (c Color) AttrValue() string {
	return string(c)
}
