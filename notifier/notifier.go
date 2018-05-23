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

/*
Package notifier provides a channel that can send update notifications.
Specifically, a notifier automatically coalesces multiple notifications
such that if a previous notification is already pending, a new notification
will not be created. This is useful in scenarios like formatting changes,
where if multiple updates come in before the first one is processed, it
is preferable to apply just the final format, ignoring the intermediate ones.
*/
package notifier

import (
	l "github.com/soumya92/barista/logging"
)

// New constructs a new notifier. It returns a func that triggers a notification,
// and a <-chan that consumes these notifications.
func New() (func(), <-chan struct{}) {
	ch := make(chan struct{}, 1)
	return func() { notify(ch) }, ch
}

func notify(ch chan<- struct{}) {
	l.Fine("Notify %s", l.ID(ch))
	select {
	case ch <- struct{}{}:
	default:
	}
}
