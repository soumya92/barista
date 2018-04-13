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
	"fmt"
	htmlTemplate "html/template"
	textTemplate "text/template"

	"github.com/dustin/go-humanize"
	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/bar"
)

var defaultFuncs = make(map[string]interface{})

// AddTemplateFunc adds template functions available to all templates.
func AddTemplateFunc(name string, f interface{}) {
	defaultFuncs[name] = f
}

// TextTemplate creates a TemplateFunc from the given text template.
func TextTemplate(tpl string) TemplateFunc {
	t := textTemplate.Must(
		textTemplate.New("text").
			Funcs(textTemplate.FuncMap(defaultFuncs)).
			Parse(tpl))
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
	t := htmlTemplate.Must(
		htmlTemplate.New("pango").
			Funcs(htmlTemplate.FuncMap(defaultFuncs)).
			Parse(tpl))
	return func(arg interface{}) bar.Output {
		var out bytes.Buffer
		if err := t.Execute(&out, arg); err != nil {
			return Error(err)
		}
		return bar.PangoSegment(out.String())
	}
}

func Bytesize(v unit.Datasize) string {
	intval := uint64(v.Bytes())
	return humanize.Bytes(intval)
}
func IBytesize(v unit.Datasize) string {
	intval := uint64(v.Bytes())
	return humanize.IBytes(intval)
}
func Byterate(v unit.Datarate) string {
	intval := uint64(v.BytesPerSecond())
	return fmt.Sprintf("%s/s", humanize.Bytes(intval))
}
func IByterate(v unit.Datarate) string {
	intval := uint64(v.BytesPerSecond())
	return fmt.Sprintf("%s/s", humanize.IBytes(intval))
}

// init adds some useful default template functions.
func init() {
	AddTemplateFunc("bytesize", Bytesize)
	AddTemplateFunc("ibytesize", IBytesize)
	AddTemplateFunc("byterate", Byterate)
	AddTemplateFunc("ibyterate", IByterate)
}
