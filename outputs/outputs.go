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

// Package outputs provides helper functions to construct bar.Outputs.
package outputs

import (
	"bytes"
	"fmt"
	htmlTemplate "html/template"
	textTemplate "text/template"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/pango"
)

// TemplateFunc is a function that takes in a single argument constructs a
// bar output from it.
type TemplateFunc func(interface{}) bar.Output

// Empty constructs an empty output, which will hide a module from the bar.
func Empty() bar.Output {
	return bar.Output{}
}

// Error constructs a bar output that indicates an error.
func Error(e error) bar.Output {
	return bar.Output{bar.NewSegment(e.Error()).
		ShortText("Error").
		Urgent(true),
	}
}

// Textf constructs simple text output from a format string and arguments.
func Textf(format string, args ...interface{}) bar.Output {
	return Text(fmt.Sprintf(format, args...))
}

//Text constructs a simple text output from the given string.
func Text(text string) bar.Output {
	return bar.Output{bar.NewSegment(text)}
}

// PangoUnsafe constructs a bar output from existing pango markup.
// This function does not perform any escaping.
func PangoUnsafe(markup string) bar.Output {
	return bar.Output{bar.NewSegment(markup).Markup(bar.MarkupPango)}
}

// Pango constructs a bar output from a list of things.
func Pango(things ...interface{}) bar.Output {
	// The extra span tag will be collapsed if no attributes were added.
	return PangoUnsafe(pango.Span(things...).Pango())
}

// TextTemplate creates a TemplateFunc from the given text template.
func TextTemplate(tpl string) TemplateFunc {
	t := textTemplate.Must(textTemplate.New("text").Parse(tpl))
	return func(arg interface{}) bar.Output {
		var out bytes.Buffer
		if err := t.Execute(&out, arg); err != nil {
			return Error(err)
		}
		return Text(out.String())
	}
}

// PangoTemplate creates a TemplateFunc from the given pango template.
// It uses go's html/template to escape input properly.
func PangoTemplate(tpl string) TemplateFunc {
	t := htmlTemplate.Must(htmlTemplate.New("pango").Parse(tpl))
	return func(arg interface{}) bar.Output {
		var out bytes.Buffer
		if err := t.Execute(&out, arg); err != nil {
			return Error(err)
		}
		return PangoUnsafe(out.String())
	}
}

// Composite represents a "composite" bar output that collects compositeple
// outputs and assigns each output a different "instance" name so that
// click handlers can know what part of the output was clicked.
type Composite interface {
	Add(string, bar.Output) Composite
	AddPango(string, ...interface{}) Composite
	AddTextf(string, string, ...interface{}) Composite
	AddText(string, string) Composite
	KeepSeparators(bool) Composite
	Build() bar.Output
}

type composite struct {
	out        bar.Output
	separators bool
}

// Add appends a named output segment to the composite bar output
// and returns it for chaining.
func (c *composite) Add(instance string, output bar.Output) Composite {
	for _, segment := range output {
		segment.Instance(instance)
		c.out = append(c.out, segment)
	}
	return c
}

// addOne adds the first (and only) element of the bar.Output after
// setting its instance and returns the composite output for chaining.
func (c *composite) addOne(instance string, output bar.Output) Composite {
	segment := output[0]
	segment.Instance(instance)
	c.out = append(c.out, segment)
	return c
}

// AddPango appends a named pango output segment to the composite
// bar output and returns it for chaining.
func (c *composite) AddPango(instance string, things ...interface{}) Composite {
	return c.addOne(instance, Pango(things...))
}

// AddTextf appends a named text output segment with formatting
// to the composite bar output and returns it for chaining.
func (c *composite) AddTextf(instance string, format string, things ...interface{}) Composite {
	return c.addOne(instance, Textf(format, things...))
}

// AddText appends a named text output segment  to the composite
// bar output and returns it for chaining.
func (c *composite) AddText(instance string, text string) Composite {
	return c.addOne(instance, Text(text))
}

// KeepSeparators sets whether inter-segment separators are removed.
// By default, inter-segment separators are removed when Build is called,
// but that behaviour can be overridden by calling KeepSeparators(true).
func (c *composite) KeepSeparators(separators bool) Composite {
	c.separators = separators
	return c
}

// Build returns the built bar.Output with each segment's instance set
// to the appropriate value.
func (c *composite) Build() bar.Output {
	if c.separators {
		return c.out
	}
	for idx, segment := range c.out {
		if idx+1 == len(c.out) {
			continue
		}
		if _, ok := segment["separator"]; ok {
			continue
		}
		segment.SeparatorWidth(0)
		segment.Separator(false)
	}
	return c.out
}

// Multi creates an empty composite output, to which named segments
// can be added.
func Multi() Composite {
	return &composite{}
}
