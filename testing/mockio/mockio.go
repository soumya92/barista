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

// Package mockio provides infinite streams that can be used for testing stdin/stdout.
package mockio

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// Writable is an infinite stream that satisfies io.Writer,
// and adds methods to get portions of the output written to it.
type Writable struct {
	// The buffer that holds all non-consumed output.
	buffer bytes.Buffer
	// A channel that signals any time new output is available.
	signal chan struct{}
	// Mutex to prevent data races in buffer.
	mutex sync.Mutex
	// For simulation, if this is set the next write will return
	// this error instead of behaving normally.
	nextError error
}

// Write satisfies the io.Writer interface.
func (w *Writable) Write(out []byte) (n int, e error) {
	w.mutex.Lock()
	if w.nextError != nil {
		e = w.nextError
		w.nextError = nil
		w.mutex.Unlock()
		return
	}
	n, e = w.buffer.Write(out)
	w.mutex.Unlock()
	select {
	case w.signal <- struct{}{}:
	default:
	}
	return
}

var _ io.Writer = (*Writable)(nil)

// ReadNow clears the buffer and returns its previous contents.
func (w *Writable) ReadNow() string {
	w.mutex.Lock()
	val := w.buffer.String()
	w.buffer = bytes.Buffer{}
	w.mutex.Unlock()
	nonBlockingConsume(w.signal)
	return val
}

// ReadUntil reads up to the first occurrence of the given character,
// or until the timeout expires, whichever comes first.
func (w *Writable) ReadUntil(delim byte, timeout time.Duration) (string, error) {
	w.mutex.Lock()
	val, err := w.buffer.ReadString(delim)
	w.mutex.Unlock()
	if err == nil {
		nonBlockingConsume(w.signal)
		return val, nil
	}
	timeoutChan := time.After(timeout)
	// EOF means we ran out of bytes, so we need to wait until more are written.
	for err == io.EOF {
		select {
		case <-timeoutChan:
			return val, err
		case <-w.signal:
			var v string
			w.mutex.Lock()
			v, err = w.buffer.ReadString(delim)
			w.mutex.Unlock()
			val += v
		}
	}
	return val, err
}

// WaitForWrite waits until the timeout for a write to this stream.
func (w *Writable) WaitForWrite(timeout time.Duration) bool {
	w.mutex.Lock()
	bufLen := w.buffer.Len()
	w.mutex.Unlock()
	if bufLen != 0 {
		nonBlockingConsume(w.signal)
		return true
	}
	timeoutChan := time.After(timeout)
	select {
	case <-timeoutChan:
		return false
	case <-w.signal:
		return true
	}
}

// ShouldError sets the stream to return an error on the next write.
func (w *Writable) ShouldError(e error) {
	w.mutex.Lock()
	w.nextError = e
	w.mutex.Unlock()
}

// Stdout returns a Writable that can be used for making assertions
// against what was written to stdout.
func Stdout() *Writable {
	return &Writable{
		signal: make(chan struct{}, 1),
	}
}

// Readable is an infinite stream that satisfies io.Reader and io.Writer
// Reads block until something is written to the stream, which mimics stdin.
type Readable struct {
	// The buffer that holds all non-consumed output.
	buffer bytes.Buffer
	// A channel that signals any time new output is available.
	available chan struct{}
	// A channel that signals any time output is consumed.
	consumed chan struct{}
	// Mutex to prevent data races in buffer.
	mutex sync.Mutex
	// For simulation, if this is set the next read will return
	// this error instead of behaving normally.
	nextError error
}

// Read satisfies the io.Reader interface.
func (r *Readable) Read(out []byte) (n int, e error) {
	r.mutex.Lock()
	if r.nextError != nil {
		e = r.nextError
		r.nextError = nil
		r.mutex.Unlock()
		return
	}
	len := r.buffer.Len()
	if len == 0 {
		r.mutex.Unlock()
		<-r.available
		r.mutex.Lock()
		if r.nextError != nil {
			e = r.nextError
			r.nextError = nil
			r.mutex.Unlock()
			if len == 0 {
				r.consumed <- struct{}{}
			}
			return
		}
	}
	n, e = r.buffer.Read(out)
	r.mutex.Unlock()
	if len == 0 {
		// len == 0 means that we got a signal from available.
		// Which means that signalWrite() is now waiting
		// for a signal on consumed.
		r.consumed <- struct{}{}
	}
	if e == io.EOF {
		e = nil
	}
	return
}

// Write satisfies the io.Writer interface.
func (r *Readable) Write(out []byte) (n int, e error) {
	r.mutex.Lock()
	n, e = r.buffer.Write(out)
	r.mutex.Unlock()
	r.signalWrite()
	return
}

var _ io.Reader = (*Readable)(nil)
var _ io.Writer = (*Readable)(nil)

// WriteString proxies directly to the byte buffer but adds a signal.
func (r *Readable) WriteString(s string) (n int, e error) {
	r.mutex.Lock()
	n, e = r.buffer.WriteString(s)
	r.mutex.Unlock()
	r.signalWrite()
	return
}

// ShouldError sets the stream to return an error on the next read.
func (r *Readable) ShouldError(e error) {
	r.mutex.Lock()
	r.nextError = e
	r.mutex.Unlock()
	r.signalWrite()
}

// signalWrite signals that data was written, and waits for it to be consumed.
func (r *Readable) signalWrite() {
	select {
	case r.available <- struct{}{}:
		<-r.consumed
	default:
	}
}

// Stdin returns a Readable that can be used in place of stdin for testing.
func Stdin() *Readable {
	return &Readable{
		buffer:    bytes.Buffer{},
		available: make(chan struct{}),
		consumed:  make(chan struct{}),
	}
}

// nonBlockingConsume consumes a value from the channel if it is available.
// Useful to ensure the signal buffered channel is empty, so that false positive
// signals are eliminated.
func nonBlockingConsume(ch <-chan struct{}) {
	select {
	case <-ch:
	default:
	}
}
