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

// +build debuglog

package logging

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchrcom/testify/assert"
)

type Fooer interface {
	Foo() int
}

type intFooer int

func (i intFooer) Foo() int {
	return int(i)
}

type structFooer struct{ int }

func (s *structFooer) Foo() int {
	return s.int
}

type embedded struct {
	Fooer
	bar int
}

func (e embedded) Foo() int {
	return e.bar + e.Fooer.Foo()
}

type astruct struct {
	embedded
	str   string
	num   int
	fooer Fooer
}

// To test shortening when the type name matches the package name.
// Usually such matches indicate that the package is primarily
// intended to support this type, so shortening it from 'foo.foo'
// to just 'foo' makes sense.
type logging struct {
	foo int
}

var namedStruct = astruct{
	embedded{intFooer(4), 5},
	"baz", 8, intFooer(11),
}
var anonStruct = struct {
	float64
	foo int
}{3.14, 15}

var testingT assert.TestingT

var floatVal = 4.5
var stringVal = "astring"
var emptyStruct = struct{}{}

var boolChan = make(chan bool)
var structChan = make(chan astruct)
var emptyChan = make(chan struct{})

var fooer42 = intFooer(42)
var fooer27 = &structFooer{27}

var fooerIntf Fooer = intFooer(22)

var stringSlice = []string{}
var structMap = map[struct {
	int
	float32
}]astruct{}

var namedStructRef *astruct = &namedStruct
var namedStruct1 = astruct{}
var namedStruct2 = astruct{}
var namedStruct1Ref = &namedStruct1
var namedStruct3 = namedStruct2

var idStruct = logging{}
var idStructRef = &idStruct
var idStructNewRef = &logging{}

var emptyChanSend chan<- struct{} = emptyChan
var boolChanSend chan<- bool = boolChan
var boolChanRecv <-chan bool = boolChan

var newBoolChan = make(chan bool)

func TestIdentify(t *testing.T) {
	resetLoggingState()
	testingT = t

	anonMap := map[chan bool]struct {
		int
		float64
	}{}
	anonMap2 := anonMap

	idTests := []struct {
		thing    interface{}
		expected string
	}{
		{4, "int@?"},
		{3 + 5i, "complex128@?"},
		{struct{}{}, "{}@?"},
		{&stringVal, fmt.Sprintf("string@%x", unsafe.Pointer(&stringVal))},
		{&emptyStruct, fmt.Sprintf("{}@%x", unsafe.Pointer(&emptyStruct))},
		{nil, "nil@?"},
		{emptyChanSend, fmt.Sprintf(
			"chan@%x", reflect.ValueOf(emptyChan).Pointer())},
		{&boolChanRecv, fmt.Sprintf(
			"chan bool@%x", reflect.ValueOf(boolChan).Pointer())},
		{testingT, fmt.Sprintf("testing.T@%x", unsafe.Pointer(t))},
		{&testingT, fmt.Sprintf("testing.T@%x", unsafe.Pointer(t))},
		{&(&namedStruct).embedded, fmt.Sprintf(
			"bar:logging.embedded@%x", unsafe.Pointer(&namedStruct.embedded))},
		{&namedStructRef, fmt.Sprintf(
			"bar:logging.astruct@%x", unsafe.Pointer(&namedStruct))},
		{&anonStruct, fmt.Sprintf(
			"{float64; foo int}@%x", unsafe.Pointer(&anonStruct))},
		{anonMap2, fmt.Sprintf(
			"[chan bool]{int; float64}@%x", reflect.ValueOf(anonMap).Pointer())},
		{stringSlice, fmt.Sprintf(
			"[]string@%x", reflect.ValueOf(stringSlice).Pointer())},
		{idStruct, "bar:logging@?"},
		{idStructRef, fmt.Sprintf(
			"bar:logging@%x", reflect.ValueOf(idStructRef).Pointer())},
	}

	for _, tc := range idTests {
		assert.Equal(t, tc.expected, identify(tc.thing).String(), "identify(%+#v)", tc.thing)
	}
}

func TestID(t *testing.T) {
	resetLoggingState()
	testingT = t

	idTests := []struct {
		thing    interface{}
		expected string
	}{
		{4, "int@?"},
		{"foobar", "string@?"},
		{&stringVal, "string#0"},
		{&emptyStruct, "{}#0"},
		{boolChanSend, "chan bool#0"},
		{boolChanRecv, "chan bool#0"},
		{boolChan, "chan bool#0"},
		{newBoolChan, "chan bool#1"},
		{&namedStruct, "bar:logging.astruct#0"},
		{&namedStruct1, "bar:logging.astruct#1"},
		{&namedStruct2, "bar:logging.astruct#2"},
		{&namedStruct1Ref, "bar:logging.astruct#1"},
		{&namedStruct3, "bar:logging.astruct#3"},
		{&fooer42, "bar:logging.intFooer#0"},
		{&namedStruct.embedded.bar, "int#0"},
		{fooer27, "bar:logging.structFooer#0"},
		{testingT, "testing.T#0"},
		{t, "testing.T#0"},
		{&t, "testing.T#0"},
		{&fooerIntf, "bar:logging.intFooer#1"},
		{&namedStruct.Fooer, "bar:logging.intFooer#2"},
		{&fooerIntf, "bar:logging.intFooer#1"},
		{&namedStructRef.Fooer, "bar:logging.intFooer#2"},
		{assert.New(t), "github.com/stretchrcom/testify/assert.Assertions#0"},
		{idStruct, "bar:logging@?"},
		{idStructRef, "bar:logging#0"},
		{idStructNewRef, "bar:logging#1"},
	}

	for _, tc := range idTests {
		assert.Equal(t, tc.expected, ID(tc.thing), "ID(%+#v)", tc.thing)
	}
}

func TestLabel(t *testing.T) {
	resetLoggingState()
	fineLogModules = append(fineLogModules, "bar:logging")
	testingT = t

	Label(fooer27, "27")
	assertLogged(t, "bar:logging.structFooer#0 -> bar:logging.structFooer#0<27>")
	Labelf(&fooer42, "%d", 42)
	assertLogged(t, "bar:logging.intFooer#0 -> bar:logging.intFooer#0<42>")
	Label(&stringVal, stringVal)
	assertLogged(t, "string#0 -> string#0<astring>")
	Label(floatVal, "float")
	assertLogged(t, "Cannot add identifier 'float' to float64@?")

	Attach(namedStructRef, &namedStructRef.num, ".num")
	Label(namedStructRef, "named")
	Label(namedStruct1Ref, "a")
	Attach(&namedStruct1, &namedStruct1.num, "->num")
	Label(&namedStruct2, "b")
	Label(&namedStruct3, "c")
	Label(boolChanSend, "sender")
	Label(testingT, "testingT")
	// Drain logs.
	mockStderr.ReadNow()

	labelTests := []struct {
		thing    interface{}
		expected string
	}{
		{&fooer27, "bar:logging.structFooer#0<27>"},
		{&fooer42, "bar:logging.intFooer#0<42>"},
		{&namedStruct, "bar:logging.astruct#0<named>"},
		{namedStructRef, "bar:logging.astruct#0<named>"},
		{&namedStructRef, "bar:logging.astruct#0<named>"},
		{&namedStruct.num, "bar:logging.astruct#0<named>.num"},
		{&namedStruct1, "bar:logging.astruct#1<a>"},
		{&namedStruct1.num, "bar:logging.astruct#1<a>->num"},
		{&namedStruct2, "bar:logging.astruct#2<b>"},
		{&namedStruct3, "bar:logging.astruct#3<c>"},
		{&stringVal, "string#0<astring>"},
		{&floatVal, "float64#0"},
		{&floatVal, "float64#0"},
		{boolChanRecv, "chan bool#0<sender>"}, // channels are linked.
		{t, "testing.T#0<testingT>"},
		{mockStderr, "bar:testing/mockio.Writable#0"},
	}

	for _, tc := range labelTests {
		assert.Equal(t, tc.expected, ID(tc.thing), "Labelled: ID(%+#v)", tc.thing)
	}

	Label(fooer27, "28")
	assertLogged(t, "bar:logging.structFooer#0<27> -> bar:logging.structFooer#0<28>")
	Labelf(namedStruct1Ref, "new-%s", "a")
	assertLogged(t, "bar:logging.astruct#1<a> -> bar:logging.astruct#1<new-a>")

	labelTests = []struct {
		thing    interface{}
		expected string
	}{
		{&fooer27, "bar:logging.structFooer#0<28>"},
		{&namedStruct1, "bar:logging.astruct#1<new-a>"},
		{&namedStruct1.num, "bar:logging.astruct#1<new-a>->num"},
	}

	for _, tc := range labelTests {
		assert.Equal(t, tc.expected, ID(tc.thing), "Re-Labelled: ID(%+#v)", tc.thing)
	}
}

func TestAttach(t *testing.T) {
	resetLoggingState()
	assertName := func(thing interface{}, name string) {
		assert.Equal(t, name, ID(thing), "ID(%+#v)", thing)
	}

	Attach(&namedStruct, &namedStruct.str, ".str")
	assertName(&namedStruct.str, "bar:logging.astruct#0.str")

	Attach(&namedStruct, &namedStruct.Fooer, ".Fooer")
	assertName(&namedStructRef.Fooer, "bar:logging.astruct#0.Fooer")

	Attachf(&namedStructRef.embedded.bar, &namedStruct, "->%s", "cycle")
	assertName(namedStructRef, "int#0->cycle")

	Attach(&namedStruct, &namedStruct.embedded, ".embedded")
	assertName(&namedStructRef.embedded, "int#0->cycle.embedded")

	Attach(&namedStructRef.embedded, &namedStructRef.embedded.bar, ".bar")
	assertName(&namedStructRef.embedded.bar, "int#0")
	assertLogged(t, "int#0->cycle.embedded is a descendant of int#0, cannot also be parent")

	Attach(nil, namedStruct1Ref, "rootNamedStruct")
	assertName(namedStruct1Ref, "rootNamedStruct")

	Attach(nil, &namedStructRef.str, "rootStr")
	assertName(&namedStructRef.str, "int#0->cycle.str")
	assertLogged(t, "Cannot reparent int#0->cycle.str, already attached to int#0->cycle")

	Attach(&anonStruct, anonStruct.foo, ".foo")
	assertName(anonStruct.foo, "int@?")
	assertLogged(t, "Cannot identify {float64; foo int}#0->int@?")

	Attachf(anonStruct, &anonStruct.foo, "=%d", anonStruct.foo)
	assertName(&anonStruct.foo, "int#1")
	assertLogged(t, "Cannot identify {float64; foo int}@?->int#1")

	Attach(nil, &namedStructRef.embedded.bar, "newint")
	assertName(namedStructRef, "newint->cycle")
	assertName(&namedStructRef.embedded, "newint->cycle.embedded")
}

func TestRegister(t *testing.T) {
	resetLoggingState()
	assertName := func(thing interface{}, name string) {
		assert.Equal(t, name, ID(thing), "ID(%+#v)", thing)
	}

	Attach(nil, namedStructRef, "ns") // for shorter names.
	Register(namedStructRef, "embedded", "str", "fooer")
	assertName(&namedStructRef.embedded, "ns.embedded")
	assertName(&namedStructRef.str, "ns.str")
	assertName(&namedStructRef.num, "int#0")
	assertName(&namedStructRef.fooer, "ns.fooer")

	Register(&namedStructRef.embedded, "Fooer", "bar", "baz")
	assertName(&namedStructRef.embedded.bar, "ns.embedded.bar")
	assertName(&namedStructRef.embedded.Fooer, "ns.embedded.Fooer")
	assertLogged(t, "Could not find baz in ns.embedded")

	Register(&stringVal, "length")
	assertName(&stringVal, "string#1") // #0 is ns.str.
	assertLogged(t, "Ignoring Register(...) for non-struct %+#v", &stringVal)

	Register(anonStruct, "foo")
	assertLogged(t, "Ignoring unaddressable value {float64; foo int}@?")

	Attach(&namedStruct1Ref.embedded.bar, namedStruct1Ref, "-cycle")
	Register(namedStruct1Ref, "embedded")
	Register(&namedStruct1Ref.embedded, "bar")
	assertName(&namedStruct1Ref.embedded.bar, "int#2")
	assertLogged(t, "Skipping int#2-cycle.embedded->bar, is an ancestor of int#2-cycle.embedded!")
}
