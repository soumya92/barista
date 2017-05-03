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

package mockio

import (
	"io"
	"testing"
	"time"

	"github.com/stretchrcom/testify/assert"
)

func TestStdout(t *testing.T) {
	stdout := Stdout()

	assert.Empty(t, stdout.ReadNow(), "starts empty")

	_, err := stdout.ReadUntil('x', 1*time.Millisecond)
	assert.Equal(t, io.EOF, err, "EOF when timeout expires")

	io.WriteString(stdout, "te")
	io.WriteString(stdout, "st")

	val, err := stdout.ReadUntil('s', 1*time.Millisecond)
	assert.Nil(t, err, "no error when multiple writes")
	assert.Equal(t, "tes", val, "read joins output from multiple writes")
	assert.Equal(t, "t", stdout.ReadNow(), "remaining string after ReadUntil returned by ReadNow")

	wait := make(chan *interface{})

	go (func(w io.Writer) {
		io.WriteString(w, "ab")
		io.WriteString(w, "cdef")
		wait <- nil
	})(stdout)

	<-wait
	val, err = stdout.ReadUntil('c', 1*time.Millisecond)
	assert.Nil(t, err, "no error when multiple writes in goroutine")
	assert.Equal(t, "abc", val, "read joins output from multiple writes in goroutine")
	assert.Equal(t, "def", stdout.ReadNow(), "remaining string after ReadUntil returned by ReadNow")

	go (func(w io.Writer) {
		io.WriteString(w, "ab")
		wait <- nil
		<-wait
		io.WriteString(w, "cd")
		wait <- nil
	})(stdout)

	<-wait
	val, err = stdout.ReadUntil('d', 1*time.Millisecond)
	assert.Equal(t, io.EOF, err, "EOF when delimiter write does not happen within timeout")
	assert.Equal(t, "ab", val, "returns content written before timeout")

	wait <- nil
	<-wait
	assert.Equal(t, "cd", stdout.ReadNow(), "continues normally after timeout")

	go (func(w io.Writer) {
		<-wait
		io.WriteString(w, "ab")
		time.Sleep(20 * time.Millisecond)
		io.WriteString(w, "cd")
		time.Sleep(20 * time.Millisecond)
		io.WriteString(w, "ef")
		time.Sleep(20 * time.Millisecond)
		io.WriteString(w, "gh")
		time.Sleep(20 * time.Millisecond)
		io.WriteString(w, "ij")
		wait <- nil
	})(stdout)

	wait <- nil
	val, err = stdout.ReadUntil('i', 50*time.Millisecond)
	assert.Equal(t, io.EOF, err, "EOF when delimiter write does not happen within timeout")
	assert.Equal(t, "abcdef", val, "returns content written before timeout")

	<-wait
	assert.Equal(t, "ghij", stdout.ReadNow(), "subsequent readnow returns content after timeout")
}

func TestStdin(t *testing.T) {
	stdin := Stdin()

	type readResult struct {
		contents string
		err      error
	}

	request := make(chan int)
	result := make(chan readResult)

	go (func(reader io.Reader, result chan<- readResult) {
		for r := range request {
			out := make([]byte, r)
			count, err := reader.Read(out)
			result <- readResult{string(out)[:count], err}
		}
	})(stdin, result)

	request <- 1
	select {
	case <-result:
		assert.Fail(t, "read should not return when nothing has been written")
	case <-time.After(1 * time.Millisecond):
	}

	stdin.WriteString("")
	r := <-result
	assert.Equal(t, readResult{"", nil}, r, "read returns empty string if written")

	stdin.WriteString("test")
	request <- 2
	select {
	case r := <-result:
		assert.Equal(t, readResult{"te", nil}, r, "read returns only requested content when more is available")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "read should not time out when content is available")
	}

	request <- 2
	select {
	case r := <-result:
		assert.Equal(t, readResult{"st", nil}, r, "read returns leftover on subsequent call")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "read should not time out when content is available")
	}

	stdin.WriteString("abcd")
	request <- 10
	select {
	case r := <-result:
		assert.Equal(t, readResult{"abcd", nil}, r, "read returns partial content if available")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "read should not time out when returning partial content")
	}

	stdin.WriteString("12")
	stdin.WriteString("34")
	stdin.WriteString("56")
	stdin.WriteString("78")
	request <- 8
	select {
	case r := <-result:
		assert.Equal(t, readResult{"12345678", nil}, r, "read returns concatenation of multiple writes")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "read should not time out when concatenating")
	}

	request <- 4
	select {
	case <-result:
		assert.Fail(t, "read should wait for a write when buffer has been emptied")
	case <-time.After(1 * time.Millisecond):
	}

	stdin.WriteString("xyz")
	stdin.WriteString("abc")
	select {
	case r := <-result:
		assert.Equal(t, readResult{"xyz", nil}, r, "read returns contents of first write (does not wait)")
	case <-time.After(1 * time.Millisecond):
		assert.Fail(t, "read does not time out when returning partial content")
	}

	request <- 1
	r = <-result
	assert.Equal(t, readResult{"a", nil}, r, "remaining writes are read by later requests")

	request <- 10
	r = <-result
	assert.Equal(t, readResult{"bc", nil}, r, "remaining writes are read by later requests")
}
