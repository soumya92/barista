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
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/martinlindhe/unit"

	"github.com/soumya92/barista/bar"
)

var defaultFuncs = make(template.FuncMap)

// AddTemplateFunc adds template functions available to all templates.
func AddTemplateFunc(name string, f interface{}) {
	defaultFuncs[name] = f
}

// TextTemplate returns a function that applies the given text template.
func TextTemplate(tpl string) func(interface{}) bar.Output {
	t := template.Must(
		template.New("text").
			Funcs(template.FuncMap(defaultFuncs)).
			Parse(tpl))
	return func(arg interface{}) bar.Output {
		var out bytes.Buffer
		if err := t.Execute(&out, arg); err != nil {
			return Error(err)
		}
		return bar.TextSegment(out.String())
	}
}

// Bytesize formats a Datasize in SI units using go-humanize.
// e.g. Bytesize(10 * unit.Megabyte) == "10 MB"
func Bytesize(v unit.Datasize) string {
	intval := uint64(v.Bytes())
	return humanize.Bytes(intval)
}

// IBytesize formats a Datasize in IEC units using go-humanize.
// e.g. IBytesize(10 * unit.Mebibyte) == "10 MiB"
func IBytesize(v unit.Datasize) string {
	intval := uint64(v.Bytes())
	return humanize.IBytes(intval)
}

// Byterate formats a Datarate in SI units using go-humanize.
// e.g. Byterate(10 * unit.MegabytePerSecond) == "10 MB/s"
func Byterate(v unit.Datarate) string {
	intval := uint64(v.BytesPerSecond())
	return fmt.Sprintf("%s/s", humanize.Bytes(intval))
}

// IByterate formats a Datarate in IEC units using go-humanize.
// e.g. Byterate(10 * unit.MebibytePerSecond) == "10 MiB/s"
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
