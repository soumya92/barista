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

package sink

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/outputs"
)

func TestNewSink(t *testing.T) {
	ch, s := New()
	go s(outputs.Text("foo"))

	out := <-ch
	require.Equal(t, "foo", out.Segments()[0].Text())

	require.False(t, s.Error(nil))
	select {
	case <-ch:
		require.Fail(t, "unexpected output on channel")
	default:
		// test passed
	}

	doneChan := make(chan struct{})
	go func() {
		require.True(t, s.Error(io.EOF))
		doneChan <- struct{}{}
	}()
	select {
	case out := <-ch:
		require.Error(t, out.Segments()[0].GetError())
	case <-doneChan:
		require.Fail(t, "expected error output on channel")
	}
}

func TestBufferedSink(t *testing.T) {
	ch, s := Buffered(5)
	s(outputs.Text("foo"))
	s(outputs.Text("bar"))

	out := <-ch
	require.Equal(t, "foo", out.Segments()[0].Text())

	out = <-ch
	require.Equal(t, "bar", out.Segments()[0].Text())
}

func TestNullSink(t *testing.T) {
	n := Null()
	doneChan := make(chan bool)
	go func(done chan<- bool) {
		for i := 0; i < 1000; i++ {
			n.Output(outputs.Text("foo"))
		}
		done <- true
	}(doneChan)

	select {
	case <-doneChan:
		// test passed.
	case <-time.After(time.Second):
		require.Fail(t, "Null sink failed to dump 1000 entries in 1s")
	}
}
