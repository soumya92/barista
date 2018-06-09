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
 pango.New(
   pango.Text("Red "),
   pango.Text("Bold Text").Bold()).
 Color(colors.Hex("#ff0000"))

or:
 pango.Text("Red ").
   Color(colors.Hex("#ff0000")).
   Append(pango.Text("Bold Text").Bold())
*/
package pango

import (
	"bytes"
	"fmt"
	"html"

	"github.com/soumya92/barista/bar"
)

type nodeType int

const (
	// ntElement is an element node with attributes and/or children.
	ntElement nodeType = iota
	// ntText is a text node with no markup or children.
	ntText
	// ntSizer is a <big> or <small> tag. It has no attributes,
	// and must be the only child of its parent.
	// It exists to support calls like:
	//   Text("x").Size(10.0).Smaller().Smaller().AppendText("y")
	// which would otherwise produce:
	//   <span size="smaller">xy</span>
	// but should actually produce:
	//   <span size="10240"><small><small>xy</small></small></span>
	ntSizer
)

// Node represents a node in a pango "document".
type Node struct {
	nodeType nodeType
	// For element nodes, this holds the tag name ("" = 'markup' node).
	// For text nodes, this holds the text content.
	content    string
	children   []*Node
	attributes map[string]string
}

// Append adds one or more nodes as children of the current node.
// The new nodes will inherit styles by virtue of being descendants,
// to insert them *adjacent* to the current node, use .Parent().Append(...).
func (n *Node) Append(nodes ...*Node) *Node {
	var insertPoint = n
	for len(insertPoint.children) == 1 &&
		insertPoint.children[0].nodeType == ntSizer {
		insertPoint = insertPoint.children[0]
	}
	for _, node := range nodes {
		if node.nodeType == ntElement && node.content == "" {
			// Collapse empty element nodes when appending, to reduce nesting.
			insertPoint.children = append(insertPoint.children, node.children...)
		} else {
			insertPoint.children = append(insertPoint.children, node)
		}
	}
	return n
}

// AppendText is a shortcut for Append(pango.Text(...), pango.Text(...), ...)
func (n *Node) AppendText(texts ...string) *Node {
	nodes := make([]*Node, len(texts))
	for i, t := range texts {
		nodes[i] = &Node{nodeType: ntText, content: t}
	}
	return n.Append(nodes...)
}

// AppendTextf is a shortcut for Append(pango.Textf(...))
func (n *Node) AppendTextf(format string, args ...interface{}) *Node {
	return n.Append(&Node{
		nodeType: ntText,
		content:  fmt.Sprintf(format, args...),
	})
}

// Parent creates a new node wrapping the current node,
// and returns the new parent node for further operations.
//
// For example,
//   Text("c").Condensed().Color(red).Parent().AppendText("foo").UnderlineError()
// will create
//   <span underline='error'><span stretch='condensed' color='#ff0000'>c</span>foo</span>
// where the appended "foo" is not condensed or red, and everything is underlined.
func (n *Node) Parent() *Node {
	existingNode := *n
	n.nodeType = ntElement
	n.attributes = nil
	n.content = ""
	n.children = []*Node{&existingNode}
	return n
}

// Pango returns a pango-formatted version of the node.
func (n *Node) Pango() string {
	if n.nodeType == ntText {
		return html.EscapeString(n.content)
	}
	var out bytes.Buffer
	if n.content != "" {
		out.WriteString("<")
		out.WriteString(n.content)
		for attrName, attrVal := range n.attributes {
			out.WriteString(" ")
			out.WriteString(attrName)
			out.WriteString("='")
			out.WriteString(html.EscapeString(attrVal))
			out.WriteString("'")
		}
		out.WriteString(">")
	}
	for _, c := range n.children {
		out.WriteString(c.Pango())
	}
	if n.content != "" {
		out.WriteString("</")
		out.WriteString(n.content)
		out.WriteString(">")
	}
	return out.String()
}

// Segments implements bar.Output for a single pango Node.
func (n *Node) Segments() []*bar.Segment {
	return []*bar.Segment{bar.PangoSegment(n.Pango())}
}

// New constructs a markup node that wraps the given Nodes.
func New(children ...*Node) *Node {
	return &Node{nodeType: ntElement, children: children}
}

// Text constructs a text node.
func Text(s string) *Node {
	return New(&Node{nodeType: ntText, content: s})
}

// Textf constructs a text node by interpolating arguments.
// Note that it will escape both the format string and arguments,
// so you should use pango constructs to add formatting.
// i.e.,
//  Textf("<span color='%s'>%s</span>", "red", "text")
// won't give you red text.
func Textf(format string, args ...interface{}) *Node {
	return Text(fmt.Sprintf(format, args...))
}
