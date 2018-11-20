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

package dbus

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBusTypes(t *testing.T) {
	require.NotPanics(t, func() { Session() }, "session bus")
	if os.Getenv("CI") != "true" {
		require.NotPanics(t, func() { System() }, "system bus")
	}

	// To allow -count > 1 tests.
	testBusInstance = atomic.Value{}

	require.Panics(t, func() { Test() }, "test bus before setup")
	SetupTestBus()
	require.NotPanics(t, func() { Test() }, "test bus after setup")

	require.Panics(t, func() { connect(nil, errors.New("something")) })
}

func TestExpandAndShorten(t *testing.T) {
	require := require.New(t)

	require.Equal("com.example.service.Method",
		expand("com.example.service", "Method"))
	require.Equal("com.example.service.Method.SubMethod",
		expand("com.example.service", ".Method.SubMethod"))
	require.Equal("net.example.service.Method",
		expand("com.example.service", "net.example.service.Method"))

	require.Equal("Method",
		shorten("com.example.service", "com.example.service.Method"))
	require.Equal(".Method.SubMethod",
		shorten("com.example.service", "com.example.service.Method.SubMethod"))
	require.Equal("net.example.service.Method",
		shorten("com.example.service", "net.example.service.Method"))
	require.Equal("com.example.service2.Method",
		shorten("com.example.service", "com.example.service2.Method"))
}

func TestMakeDBusName(t *testing.T) {
	require := require.New(t)

	require.Equal(dbusName{"com.example.foo", "Service"},
		makeDbusName("com.example.foo.Service"))
	require.Equal(dbusName{"com.example", "foo"},
		makeDbusName("com.example.foo"))
	require.Equal(dbusName{"com", "example"},
		makeDbusName("com.example"))
	require.Equal(dbusName{"", "example"},
		makeDbusName("example"))

	for _, s := range []string{
		"com.example.foo.Service",
		"com.example.foo",
		"com.example",
	} {
		require.Equal(s, makeDbusName(s).String(),
			"%s -> dbus -> string != %s", s, s)
	}
}
