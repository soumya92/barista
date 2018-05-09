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
	"sync/atomic"

	"github.com/soumya92/barista/bar"
)

// SimpleClickHandler allows modules to drop-in support for clicks.
// Its zero value is a nop click listener, so modules need to simply
// add an anonymous instance to their backing struct.
type SimpleClickHandler struct{ atomic.Value }

// OnClick sets the click handler.
func (s *SimpleClickHandler) OnClick(handler func(bar.Event)) {
	if handler == nil {
		handler = func(e bar.Event) {}
	}
	s.Store(handler)
}

// Click handles click events.
func (s *SimpleClickHandler) Click(e bar.Event) {
	if handler, ok := s.Load().(func(bar.Event)); ok {
		handler(e)
	}
}

// SimpleClickHandlerModule is a bar.Module and bar.Clickable that supports
// setting the click handler via OnClick(func(bar.Event)).
type SimpleClickHandlerModule interface {
	bar.Module
	bar.Clickable
	OnClick(func(bar.Event))
}
