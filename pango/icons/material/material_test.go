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

package material

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"

	"github.com/soumya92/barista/pango"
	"github.com/soumya92/barista/testing/githubfs"
	pangoTesting "github.com/soumya92/barista/testing/pango"
)

func TestInvalid(t *testing.T) {
	fs = afero.NewMemMapFs()
	assert.Error(t, Load("/src/no-such-directory"))

	afero.WriteFile(fs, "/src/material-error-1/iconfont/codepoints", []byte(
		`-- Lines in weird formats --
		 Empty:

		 Valid line:
		 someIcon 61
		 otherIcon 62
		 Invalid codepoint:
		 badIcon xy`,
	), 0644)
	assert.Error(t, Load("/src/material-error-1"))

	afero.WriteFile(fs, "/src/material-error-2/iconfont/codepoint", nil, 0644)
	assert.Error(t, Load("/src/material-error-2"))

	afero.WriteFile(fs, "/src/material-error-3/iconfont/codepoints", []byte(
		`someIcon 61
		 otherIcon 62
		 badIcon xy`,
	), 0644)
	assert.Error(t, Load("/src/material-error-3"))
}

func TestValid(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/material/iconfont/codepoints", []byte(
		`someIcon 61
		 otherIcon 62
		 thirdIcon 63`,
	), 0644)
	assert.NoError(t, Load("/src/material"))
	pangoTesting.AssertText(t, "a", pango.Icon("material-someIcon").String())
}

// TestLive tests that current master branch of the icon font works with
// this package. This test only runs when CI runs tests in 'cron' mode,
// which provides timely notifications of incompatible changes while
// keeping default tests hermetic.
func TestLive(t *testing.T) {
	if evt := os.Getenv("TRAVIS_EVENT_TYPE"); evt != "cron" {
		t.Skipf("Skipping LiveVersion test for event type '%s'", evt)
	}
	fs = githubfs.New()
	assert.NoError(t, Load("/google/material-design-icons/master"))
}
