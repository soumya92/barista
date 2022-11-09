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

// +build baristadebuglog

package logging

import (
	"log"
	"os"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type args []interface{}

func invoke(method interface{}, args args) {
	methodV := reflect.Indirect(reflect.ValueOf(method))
	values := make([]reflect.Value, len(args))
	for i, arg := range args {
		values[i] = reflect.ValueOf(arg)
	}
	methodV.Call(values)
}

func TestRace(t *testing.T) {
	resetLoggingState()
	fineLogModules = []string{"bar:logging"}

	methods := []struct {
		method interface{}
		args   args
	}{
		{Log, args{"Test: %d %s", 4, "string"}},
		{Log, args{"Test: %s", "a"}},
		{Log, args{"Test: %s", "b"}},
		{Log, args{"Test: %s", "c"}},
		{Log, args{"Test: %s", "d"}},
		{Fine, args{"Fine: %s %g", "float", 2.718}},
		{Fine, args{"Fine: %s", "a"}},
		{Fine, args{"Fine: %s", "b"}},
		{Fine, args{"Fine: %s", "c"}},
		{Fine, args{"Fine: %s", "d"}},
		{SetFlags, args{log.Lshortfile | log.Ltime}},
		{SetFlags, args{log.Llongfile}},
		{SetOutput, args{os.Stderr}},
		{SetOutput, args{os.Stdout}},
		{ID, args{4}},
		{ID, args{3 + 5i}},
		{ID, args{&boolChan}},
		{ID, args{boolChanSend}},
		{ID, args{boolChanRecv}},
		{Attach, args{&namedStruct1.embedded, &namedStruct1.embedded.bar, ".bar"}},
		{Attachf, args{namedStruct1Ref, &namedStruct1.embedded, "->%s", "e"}},
		{Attach, args{&namedStructRef.embedded, &namedStructRef.embedded.Fooer, "->Fooer"}},
		{Register, args{&namedStructRef, "embedded", "blah"}},
		{Attach, args{Root, &namedStructRef, "ns"}},
		{Label, args{&namedStruct2, "b"}},
		{Label, args{&namedStruct2, "c"}},
		{Label, args{&namedStruct2, "d"}},
		{Label, args{&namedStruct2, "e"}},
		{Labelf, args{&namedStruct1, "STRUCT%d", 1}},
	}

	// Because we're intentionally scrambling the method order,
	// to get a deterministic number we need to call ID on the
	// objects we care about first.
	ID(&namedStruct1)

	var wg sync.WaitGroup
	wg.Add(len(methods))

	var launch = make(chan struct{})
	for _, m := range methods {
		go func(method interface{}, args args) {
			// For maximum contention, try to invoke methods at the same time.
			<-launch
			invoke(method, args)
			wg.Done()
		}(m.method, m.args)
	}
	for range methods {
		launch <- struct{}{}
	}

	wg.Wait()

	// We don't know what namedStruct2 is labelled, but re-labelling
	// should still work.
	Label(&namedStruct2, "newlabel")

	require := require.New(t)
	// Make some assertions about the generated IDs.
	require.Equal("chan bool#0", ID(boolChan))
	require.Equal("bar:logging.astruct#0<STRUCT1>->e.bar", ID(&namedStruct1.embedded.bar))
	require.Equal("ns.embedded->Fooer", ID(&namedStructRef.embedded.Fooer))
	require.Regexp(`bar:logging\.astruct#\d<newlabel>`, ID(&namedStruct2))
}
