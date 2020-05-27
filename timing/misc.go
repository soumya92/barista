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
	"time"
)

// nextAlignedExpiration calculates the next expiration time.
// The expiration is first rounded to interval granularity, and then offset is added.
//
// For example, given interval 1h, and offset 15m, the expirations will happen at
// :15 of every hour regardless of the initial time.
func nextAlignedExpiration(initial time.Time, interval time.Duration, offset time.Duration) time.Time {
	next := initial.Truncate(interval).Add(offset)
	if !next.After(initial) {
		next = next.Add(interval)
	}
	if !next.After(initial) {
		panic("nextAlignedExpiration: bug: !next.After(initial)")
	}
	return next
}
