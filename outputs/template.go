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
	"bytes"
	htmlTemplate "html/template"
	textTemplate "text/template"

	"github.com/soumya92/barista/bar"
)

// TextTemplate creates a TemplateFunc from the given text template.
func TextTemplate(tpl string) TemplateFunc {
	t := textTemplate.Must(textTemplate.New("text").Parse(tpl))
	return func(arg interface{}) bar.Output {
		var out bytes.Buffer
		if err := t.Execute(&out, arg); err != nil {
			return Error(err)
		}
		return bar.TextSegment(out.String())
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
		return bar.PangoSegment(out.String())
	}
}
