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
	"log"
	"os"
	"runtime"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/testing/mockio"
)

var mockStderr *mockio.Writable

func resetLoggingState() {
	mu.Lock()
	defer mu.Unlock()

	nodes = map[ident]node{}
	instances = map[string]int{}
	fineLogModules = []string{}
	objectIDs = map[ident]string{}
	labels = map[ident]string{}

	fineLogModulesCache.Range(func(k, v interface{}) bool {
		fineLogModulesCache.Delete(k)
		return true
	})

	construct()
	mockStderr = mockio.Stdout()
	SetFlags(0) // To make test output as deterministic as possible.
	SetOutput(mockStderr)
}

func assertLogged(t *testing.T, format string, args ...interface{}) {
	assert.Equal(t, fmt.Sprintf(format, args...)+"\n", mockStderr.ReadNow())
}

func TestShorten(t *testing.T) {
	shortenTests := []struct {
		fullpath string
		expected string
	}{
		{"github.com/soumya92/barista.Run", "barista:Run"},
		{"github.com/soumya92/barista.(*i3Bar).AddModule",
			"barista:i3Bar.AddModule"},
		{"github.com/soumya92/barista/modules/weather/darksky.(*Provider).GetWeather",
			"mod:weather/darksky.Provider.GetWeather"},
		{"github.com/soumya92/barista/modules/clock", "mod:clock"},
		{"github.com/soumya92/barista/core.Module", "core:Module"},
		{"github.com/soumya92/barista/base.(Value).Set", "bar:base.Value.Set"},
		{"github.com/golang/go/pkg.Type.Method",
			"github.com/golang/go/pkg.Type.Method"},
		{"builtin.Type.Method", "builtin.Type.Method"},
	}

	for _, tc := range shortenTests {
		assert.Equal(t, tc.expected, shorten(tc.fullpath),
			"shorten(%s)", tc.fullpath)
	}
}

func TestLog(t *testing.T) {
	resetLoggingState()
	Log("something: %s", "foo")
	assertLogged(t, "something: foo")
}

func TestFine(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()
	arg0 := os.Args[0]

	os.Args = []string{arg0}
	resetLoggingState()
	Fine("foo")
	assert.Empty(t, mockStderr.ReadNow())

	os.Args = []string{arg0, "-finelog=mod:something,"}
	resetLoggingState()
	Fine("foo")
	assertLogged(t, "foo")

	os.Args = []string{arg0, "--finelog=bar:logging.TestLog"}
	resetLoggingState()
	Fine("foo")
	assert.Empty(t, mockStderr.ReadNow())

	os.Args = []string{arg0, "--finelog=bar:"}
	resetLoggingState()
	Fine("foo")
	assertLogged(t, "foo")

	os.Args = []string{arg0, "--finelog=bar:notifier.Test,bar:logging.TestFine"}
	resetLoggingState()
	Fine("foo")
	assertLogged(t, "foo")

	os.Args = []string{arg0, "--finelog=bar:logging.Test"}
	resetLoggingState()
	Fine("foo")
	assertLogged(t, "foo")

	os.Args = []string{arg0, "--finelog=bar:colors.Test"}
	resetLoggingState()
	Fine("foo")
	assert.Empty(t, mockStderr.ReadNow())
}

func TestFileLocations(t *testing.T) {
	resetLoggingState()
	SetFlags(log.Lshortfile)
	Log("foo")
	_, _, line, _ := runtime.Caller(0)
	assertLogged(t, fmt.Sprintf("logging_test.go:%d (bar:logging.TestFileLocations) foo", line-1))
}
