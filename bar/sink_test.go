// Copyright 2017 Google Inc.
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

package bar

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutput(t *testing.T) {
	var sink Sink
	ch := make(chan Output, 1)
	sink = func(o Output) { ch <- o }

	sink.Output(TextSegment("foo"))
	select {
	case out := <-ch:
		require.Equal(t, "foo", out.Segments()[0].Text())
	default:
		require.Fail(t, "expected output on Output(...)")
	}

	sink.Output(nil)
	require.Nil(t, <-ch)
}

func TestError(t *testing.T) {
	var sink Sink
	ch := make(chan Output, 1)
	sink = func(o Output) { ch <- o }

	require.False(t, sink.Error(nil), "nil error returns false")
	select {
	case <-ch:
		require.Fail(t, "Should not send any output on Error(nil)")
	default:
		// test passed.
	}

	require.True(t, sink.Error(io.EOF), "non-nil error returns true")
	select {
	case out := <-ch:
		require.Error(t, out.Segments()[0].GetError(),
			"output sent on Error(...) has error segment")
	default:
		require.Fail(t, "Expected an error output on Error(...)")
	}
}
