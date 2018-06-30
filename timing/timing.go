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

/*
Package timing provides a testable interface for timing and scheduling.

This makes it simple to update a module at a fixed interval or
at a fixed point in time.

Typically, modules will make a scheduler:
    mod.sch = timing.NewScheduler()
and use the scheduling calls to control the update timing:
    mod.sch.Every(time.Second)

The Stream() goroutine will then loop over the ticker, and update
the module with fresh information:
    for range mod.sch.Tick() {
	  // update code.
    }

This will automatically suspend processing when the bar is hidden.

Modules should also use timing.Now() instead of time.Now() to control time
during tests.
*/
package timing

import "time"

// Now returns the current time.
var Now = time.Now
