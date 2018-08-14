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

package base

import (
	"sync"

	l "github.com/soumya92/barista/logging"
	"github.com/soumya92/barista/notifier"
)

// Emitter is a source of multicast updates. It can be created by calling
// base.Multicast(...) on a chan struct{}, e.g. base.Multicast(fooV.Update()).
// The returned emitter will then allow multiple subscriptions to fooV, and
// will notify each subscription whenever fooV.Update() sends a value.
type Emitter struct {
	subMu sync.RWMutex
	subs  []func()
	obs   []chan struct{}
}

// Multicast allows multiple listeners for a single notifying channel by
// constructing an emitter that wraps the given channel.
func Multicast(source <-chan struct{}) *Emitter {
	e := &Emitter{}
	go e.notifyOn(source)
	return e
}

// notifyOn loops and triggers all subscriptions on each update from source.
func (e *Emitter) notifyOn(source <-chan struct{}) {
	for range source {
		e.subMu.RLock()
		l.Fine("%s emit to %d+%d listener(s)", l.ID(e), len(e.subs), len(e.obs))
		for _, notifyFn := range e.subs {
			notifyFn()
		}
		for _, ob := range e.obs {
			close(ob)
		}
		e.obs = nil
		e.subMu.RUnlock()
	}
}

// Subscribe creates a new ticker associated with the emitter's source.
func (e *Emitter) Subscribe() <-chan struct{} {
	fn, ch := notifier.New()
	e.subMu.Lock()
	l.Attachf(e, ch, "$%d", len(e.subs))
	e.subs = append(e.subs, fn)
	e.subMu.Unlock()
	return ch
}

// Next creates a new channel that will be closed when the emitter's source
// updates next. This is an alternative to Subscribe() that does not need
// to be cleaned up, so can be freely used in a loop.
func (e *Emitter) Next() <-chan struct{} {
	ch := make(chan struct{})
	e.subMu.Lock()
	e.obs = append(e.obs, ch)
	e.subMu.Unlock()
	return ch
}
