// Copyright 2020 Google Inc.
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

// +build !linux

package timing

// NewRealtimeScheduler creates a scheduler backed by system real-time clock.
//
// It properly handles system suspend (sleep mode) and time adjustments. For periodic timers,
// it triggers immediately whenever time changes discontinuously. For one-shot timers
// (At and After), it will fire immediately if the time is skipped over
// the set trigger time, and will properly wait for it otherwise.
//
// This scheduler is only properly supported on Linux. On other systems,
// plain scheduler based on "time" package is returned.
//
// In order to clean up resources associated with it,
// remember to call Stop().
func NewRealtimeScheduler() (*Scheduler, error) {
	return NewScheduler(), nil
}
