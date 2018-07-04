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
	"reflect"

	"github.com/soumya92/barista/outputs"
)

// Template calls the provided Output func with a reflectively
// created function that invokes the given template.
// This avoids boilerplate code of the form:
//     func (m *Module) Template(tpl string) {
//         tplFn := outputs.TextTemplate(tpl)
//         m.Output(func(f Foo) bar.Output {
//             tplFn(f)
//         })
//     }
// by replacing it with
//     func (m *Module) Template(tpl string) {
//         base.Template(tpl, m.Output)
//     }
// This function can panic in quite a few places. So make sure that:
// - The template provided is valid (likely up to the user)
// - The second argument is a function, which takes a function.
// - The function argument takes one argument, and returns a bar.Output.
func Template(template string, OutputFunc interface{}) {
	templateFn := outputs.TextTemplate(template)
	funcType := reflect.TypeOf(OutputFunc).In(0)
	fn := reflect.MakeFunc(funcType, func(args []reflect.Value) []reflect.Value {
		out := templateFn(args[0].Interface())
		return []reflect.Value{
			reflect.ValueOf(out).Convert(funcType.Out(0)),
		}
	})
	reflect.ValueOf(OutputFunc).Call([]reflect.Value{fn})
}
