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

package base

import (
	"testing"

	"github.com/martinlindhe/unit"
	"github.com/stretchr/testify/assert"

	"github.com/soumya92/barista/bar"
)

type Data struct {
	Number  int
	String  string
	Decimal float64
}

var sampleData = Data{42, "foobar", 2.718}

var result bar.Output

func Simple(f func(Data) bar.Output) {
	result = f(sampleData)
}

func Pointer(f func(*Data) bar.Output) string {
	result = f(&sampleData)
	return "whatever"
}

func Primitive(f func(float64) bar.Output) {
	result = f(3.14159)
}

func TypedPrimitive(f func(unit.Length) bar.Output) *Data {
	result = f(1 * unit.Meter)
	return &sampleData
}

var templateTests = []struct {
	desc     string
	template string
	function interface{}
	expected string
}{
	{"simple", `{{.Number}}`, Simple, "42"},
	{"pointer", `{{.String}}`, Pointer, "foobar"},
	{"primitive", `{{. | printf "%0.2f"}}`, Primitive, "3.14"},
	{"typed primitive", `{{. | printf "%0.2f"}}`, TypedPrimitive, "1.00"},
	{"typed primitive (method)", `{{.Inches | printf "%0.1f"}}`, TypedPrimitive, "39.4"},
}

func TestTemplates(t *testing.T) {
	for _, tc := range templateTests {
		Template(tc.template, tc.function)
		assert.Equal(t, tc.expected, result.Segments()[0].Text(), tc.desc)
	}
}
