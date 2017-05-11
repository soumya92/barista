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
	return bar.Output{bar.NewSegment().
		Text(e.Error()).
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
	return bar.Output{bar.NewSegment().Text(text)}
}

// PangoUnsafe constructs a bar output from existing pango markup.
// This function does not perform any escaping.
func PangoUnsafe(markup string) bar.Output {
	return bar.Output{bar.NewSegment().
		Text(markup).
		Markup(bar.MarkupPango),
	}
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
