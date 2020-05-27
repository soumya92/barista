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

package timing

import (
	"testing"
	"time"
)

func testSchedulerImplementation(t *testing.T, create func() *Scheduler) {
	if testing.Short() {
		t.Skip("skipping scheduler tests that need real time passage in short mode.")
	}

	t.Run("At", func(t *testing.T) {
		s := create()
		defer s.Close()

		begin := time.Now()
		s.At(begin.Add(time.Second))

		select {
		case <-s.C:
			if time.Now().Sub(begin) < time.Second {
				t.Errorf("scheduler triggered too early begin=%v now=%v", begin, time.Now())
			}
		case <-time.After(2 * time.Second):
			t.Error("scheduler did not trigger")
		}

		select {
		case <-s.C:
			t.Error("scheduler triggered twice!")
		case <-time.After(2 * time.Second):
		}
	})

	t.Run("AtStop", func(t *testing.T) {
		s := create()
		defer s.Close()

		begin := time.Now()
		s.At(begin.Add(2 * time.Second))
		time.Sleep(time.Second)
		s.Stop()

		select {
		case <-s.C:
			t.Error("scheduler triggered even though stopped")
		case <-time.After(2 * time.Second):
		}
	})

	t.Run("After", func(t *testing.T) {
		s := create()
		defer s.Close()

		begin := time.Now()
		s.After(time.Second)

		select {
		case <-s.C:
			if time.Now().Sub(begin) < time.Second {
				t.Errorf("scheduler triggered too early begin=%v now=%v", begin, time.Now())
			}
		case <-time.After(2 * time.Second):
			t.Error("scheduler did not trigger")
		}
		select {
		case <-s.C:
			t.Error("scheduler triggered twice!")
		case <-time.After(2 * time.Second):
		}
	})

	t.Run("AfterStop", func(t *testing.T) {
		s := create()
		defer s.Close()

		s.After(2 * time.Second)
		time.Sleep(time.Second)
		s.Stop()

		select {
		case <-s.C:
			t.Error("scheduler triggered even though stopped")
		case <-time.After(2 * time.Second):
		}
	})

	t.Run("Every", func(t *testing.T) {
		s := create()
		defer s.Close()

		begin := time.Now()
		s.Every(time.Second)

		for i := 0; i < 2; i++ {
			select {
			case <-s.C:
				if time.Now().Sub(begin) < time.Second*time.Duration(i+1) {
					t.Errorf("scheduler triggered too early (tick %d) begin=%v now=%v", i, begin, time.Now())
				}
			case <-time.After(2 * time.Second):
				t.Errorf("scheduler did not trigger (tick %d)", i)
			}
		}
		s.Stop()
		select {
		case <-s.C:
			t.Error("scheduler triggered even though stopped")
		case <-time.After(2 * time.Second):
		}
	})

	t.Run("EveryAlign", func(t *testing.T) {
		s := create()
		defer s.Close()

		const interval = time.Second

		now := time.Now()
		time.Sleep(now.Add(interval).Truncate(interval).Add(interval / 2).Sub(now))
		// sleep until next :00.500 second

		s.EveryAlign(time.Second, time.Duration(0))

		for i := 0; i < 2; i++ {
			select {
			case <-s.C:
				now := time.Now()
				// on my machine, the alignment precision is usually 0.1ms
				if now.Sub(now.Truncate(interval)) > 10*time.Millisecond {
					t.Errorf("trigger time was not aligned (tick %d) now=%v", i, now)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("scheduler did not trigger (tick %d)", i)
			}
		}
		s.Stop()
		select {
		case <-s.C:
			t.Error("scheduler triggered even though stopped")
		case <-time.After(2 * time.Second):
		}
	})

}
