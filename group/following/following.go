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

// Package following provides a group that always shows the output from
// the most recently updated module in the set.
package following // import "barista.run/group/following"

import (
	"sync/atomic"

	"barista.run/bar"
	"barista.run/group"
)

// grouper implements a following grouper.
type grouper struct{ current int64 }

// Group returns a new following group.
func Group(m ...bar.Module) bar.Module {
	return group.New(&grouper{}, m...)
}

func (g *grouper) Visible(idx int) bool {
	return atomic.LoadInt64(&g.current) == int64(idx)
}

func (g *grouper) Updated(idx int) {
	atomic.StoreInt64(&g.current, int64(idx))
}

func (g *grouper) Buttons() (start, end bar.Output) { return nil, nil }
