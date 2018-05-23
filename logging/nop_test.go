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

// +build !debuglog

package logging

import (
	"log"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"github.com/soumya92/barista/testing/mockio"
)

func TestNopMethods(t *testing.T) {
	SetOutput(mockio.Stdout())
	SetFlags(log.Lshortfile)
	Log("foo: %d", 42)
	Fine("bar: %g", 3.14159)
	assert.Equal(t, "", ID(4))
	Label(&struct{}{}, "empty")
	Labelf(&struct{}{}, "empty: %b", true)
	Attach(t, 4, "->int")
	Attachf(t, 1.0, "->float:%g", 1.0)
	Register(t, "Fail", "FailNow")
}
