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
	"testing"

	"barista.run/pango"
	"barista.run/testing/cron"
	"barista.run/testing/githubfs"
	pangoTesting "barista.run/testing/pango"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestInvalid(t *testing.T) {
	fs = afero.NewMemMapFs()
	require.Error(t, Load("/src/no-such-directory"))

	afero.WriteFile(fs, "/src/typicons-error-1/src/font/typicons.json", []byte(
		`-- Invalid JSON --`,
	), 0644)
	require.Error(t, Load("/src/typicons-error-1"))

	afero.WriteFile(fs, "/src/typicons-error-2/src/font/typicons.json", nil, 0644)
	require.Error(t, Load("/src/typicons-error-2"))

	afero.WriteFile(fs, "/src/typicons-error-3/src/font/typicons.json", []byte(
		`{
			"someIcon": 97,
  			"otherIcon": 98,
  			"thirdIcon": "ghij"
		}
`,
	), 0644)
	require.Error(t, Load("/src/typicons-error-3"))
}

func TestValid(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/typicons/src/font/typicons.json", []byte(
		`{
			"someIcon": 97,
  			"otherIcon": 98,
  			"thirdIcon": 99
		}
`,
	), 0644)
	require.NoError(t, Load("/src/typicons"))
	pangoTesting.AssertText(t, "a", pango.Icon("typecn-someIcon").String())
	pangoTesting.AssertText(t, "b", pango.Icon("typecn-otherIcon").String())
}

// TestLive tests that current master branch of the icon font works with
// this package. This test only runs when CI runs tests in 'cron' mode,
// which provides timely notifications of incompatible changes while
// keeping default tests hermetic.
func TestLive(t *testing.T) {
	fs = githubfs.New()
	cron.Test(t, func() error {
		if err := Load("/stephenhutchings/typicons.font/master"); err != nil {
			return err
		}
		// At least one of these icons should be loaded.
		testIcons := pango.New(
			pango.Icon("typecn-pen"),
			pango.Icon("typecn-flag-outline"),
			pango.Icon("typecn-plus"),
			pango.Icon("typecn-beaker"),
		)
		require.NotEmpty(t, testIcons.String(), "No expected icons were loaded")
		return nil
	})
}
