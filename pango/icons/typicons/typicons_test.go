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

package typicons

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

	afero.WriteFile(fs, "/src/typicons-error-1/config.yml", []byte(
		`-- Invalid YAML --`,
	), 0644)
	assert.Error(t, Load("/src/typicons-error-1"))

	afero.WriteFile(fs, "/src/typicons-error-2/config.yml", nil, 0644)
	assert.Error(t, Load("/src/typicons-error-2"))

	afero.WriteFile(fs, "/src/typicons-error-3/config.yml", []byte(
		`glyphs:
- css: someIcon
  code: 0x61
- css: otherIcon
  code: 0x62
- css: thirdIcon
  code: 0xghij
`,
	), 0644)
	assert.Error(t, Load("/src/typicons-error-3"))
}

func TestValid(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/typicons/config.yml", []byte(
		`glyphs:
- css: someIcon
  code: 0x61
- css: otherIcon
  code: 0x62
- css: thirdIcon
  code: 0x63
`,
	), 0644)
	assert.NoError(t, Load("/src/typicons"))
	pangoTesting.AssertText(t, "a", pango.Icon("typecn-someIcon").String())
	pangoTesting.AssertText(t, "b", pango.Icon("typecn-otherIcon").String())
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
	assert.NoError(t, Load("/stephenhutchings/typicons.font/master"))
}
