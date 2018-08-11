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

package base

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/soumya92/barista/bar"
)

func TestClickHandler(t *testing.T) {
	var s SimpleClickHandler

	require.NotPanics(t, func() { s.Click(bar.Event{}) },
		"Without a click handler set")

	clickedChan := make(chan bool)
	s.OnClick(func(e bar.Event) {
		clickedChan <- true
	})
	go s.Click(bar.Event{})
	select {
	case <-clickedChan:
		// Test passed.
	case <-time.After(time.Second):
		require.Fail(t, "Click event not sent to handler")
	}

	require.NotPanics(t, func() { s.OnClick(nil) },
		"Clearing click handler")
	require.NotPanics(t, func() { s.Click(bar.Event{}) },
		"After removing click handler")
}
