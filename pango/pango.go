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

/*
Package pango provides a type-safe way to construct pango markup.
Using nested Span and Text nodes, pango formatted output can be easily constructed
with compile-time validation of nesting and automatic escaping.

For example, to construct pango markup for:
 <span color="#ff0000">Red <span weight="bold">Bold Text</span></span>

the go code would be:
 pango.Span(
   colors.Hex("#ff0000"),
   "Red ",
   pango.Span(
     pango.Weight("bold"),
     "Bold Text",
   ),
 )
*/
package pango

import (
	"bytes"
	"fmt"
	"html"
	"strings"
)

// Node represents nodes in a pango "document".
type Node interface {
	// Pango returns a pango-formatted version of the element.
	Pango() string
}

// Attribute represents a pango attribute name and value.
type Attribute interface {
	AttrName() string
	AttrValue() string
}

// element represents a generic element.
type element struct {
	tagName    string
	attributes []Attribute
	children   []Node
}

// collapse returns true if the element is "useless", i.e. an empty Span.
func (e *element) collapse() bool {
	return strings.EqualFold(e.tagName, "span") && len(e.attributes) == 0
}

func (e *element) Pango() string {
	printTag := !e.collapse()
	var out bytes.Buffer
	if printTag {
		out.WriteString("<")
		out.WriteString(e.tagName)
		for _, attr := range e.attributes {
			out.WriteString(" ")
			out.WriteString(attr.AttrName())
			out.WriteString("='")
			out.WriteString(html.EscapeString(attr.AttrValue()))
			out.WriteString("'")
		}
		out.WriteString(">")
	}
	for _, c := range e.children {
		out.WriteString(c.Pango())
	}
	if printTag {
		out.WriteString("</")
		out.WriteString(e.tagName)
		out.WriteString(">")
	}
	return out.String()
}

// Text represents a plaintext section of text.
type Text string

// Pango returns html-escaped text.
func (t Text) Pango() string {
	return html.EscapeString(string(t))
}

// Textf constructs a text node by interpolating arguments.
// Note that it will escape both the format string and arguments,
// so you should use pango constructs to add formatting.
// i.e.,
//  Textf("<span color='%s'>%s</span>", "red", "text")
// won't give you red text.
func Textf(format string, args ...interface{}) Node {
	return Text(fmt.Sprintf(format, args...))
}

// Tag constructs a pango element with the given name, with any children and/or attributes.
// The interface varargs are used as below:
//  - A pango.Attribute is added to the tag directly
//  - A pango.Element is added as a child node
//  - Any other object is added as a text node using the %v format.
func Tag(tagName string, things ...interface{}) Node {
	e := &element{tagName: tagName}
	for _, thing := range things {
		switch thing := thing.(type) {
		case Attribute:
			e.attributes = append(e.attributes, thing)
		case Node:
			e.children = append(e.children, thing)
		default:
			e.children = append(e.children, Textf("%v", thing))
		}
	}
	return e
}

// Span constructs a new span with the given attributes and segments.
func Span(things ...interface{}) Node {
	return Tag("span", things...)
}
