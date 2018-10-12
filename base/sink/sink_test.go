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

	"barista.run/bar"
	"barista.run/outputs"

	"github.com/stretchr/testify/require"
)

func TestNewSink(t *testing.T) {
	ch, s := New()
	go s(outputs.Text("foo"))

	out := <-ch
	txt, _ := out.Segments()[0].Content()
	require.Equal(t, "foo", txt)

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
	txt, _ := out.Segments()[0].Content()
	require.Equal(t, "foo", txt)

	out = <-ch
	txt, _ = out.Segments()[0].Content()
	require.Equal(t, "bar", txt)
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

func TestValueSink(t *testing.T) {
	v, s := Value()

	next := v.Next()
	s.Output(outputs.Text("foo"))

	<-next
	out := v.Get().(bar.Output)
	txt, _ := out.Segments()[0].Content()
	require.Equal(t, "foo", txt)

	next = v.Next()
	s.Output(nil)

	<-next
	require.Nil(t, v.Get(), "nil output")
}
