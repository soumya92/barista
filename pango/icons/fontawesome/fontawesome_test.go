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

package fontawesome

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
	var err error
	require.Error(t, Load("/src/no-such-directory"))

	afero.WriteFile(fs, "/src/fa-error-1/metadata/icons.yml", []byte(
		`-- Invalid YAML --`,
	), 0644)
	err = Load("/src/fa-error-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot unmarshal")

	afero.WriteFile(fs, "/src/fa-error-2/metadata/icons.yml", nil, 0644)
	err = Load("/src/fa-error-2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "EOF")

	afero.WriteFile(fs, "/src/fa-error-3/metadata/icons.yml", []byte(
		`some-icon:
  changes:
    - '4.4'
    - 5.0.0
  label: Should Not Matter
  styles:
    - solid
  unicode: abcd
bad-icon:
  styles:
    - regular
  unicode: ghij
`,
	), 0644)
	err = Load("/src/fa-error-3")
	require.Error(t, err)
	require.Contains(t, err.Error(), "ghij")

	afero.WriteFile(fs, "/src/fa-error-4/metadata/icons.yml", []byte(
		`some-icon:
  label: Should Not Matter
  styles:
    - unknownstyle
  unicode: abcd
`,
	), 0644)
	err = Load("/src/fa-error-4")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Unknown FontAwesome style")
}

func TestValid(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/fa/metadata/icons.yml", []byte(
		`
some-icon:
  styles:
    - solid
  unicode: 61
other-icon:
  styles:
    - regular
  unicode: 62
brand-icon:
  styles:
    - brands
  unicode: 63
all-icon:
  styles:
    - brands
    - regular
    - solid
  unicode: 64
`,
	), 0644)
	require.NoError(t, Load("/src/fa"))

	pangoTesting.AssertText(t, "a", pango.Icon("fa-some-icon").String())
	pangoTesting.AssertText(t, "b", pango.Icon("far-other-icon").String())
	pangoTesting.AssertText(t, "c", pango.Icon("fab-brand-icon").String())

	pangoTesting.AssertText(t, "d", pango.Icon("fa-all-icon").String())
	pangoTesting.AssertText(t, "d", pango.Icon("far-all-icon").String())
	pangoTesting.AssertText(t, "d", pango.Icon("fab-all-icon").String())

}

// TestLive tests that current master branch of the icon font works with
// this package. This test only runs when CI runs tests in 'cron' mode,
// which provides timely notifications of incompatible changes while
// keeping default tests hermetic.
func TestLive(t *testing.T) {
	fs = githubfs.New()
	cron.Test(t, func() error {
		if err := Load("/FortAwesome/Font-Awesome/master"); err != nil {
			return err
		}
		// At least one of these icons should be loaded.
		testIcons := pango.New(
			pango.Icon("fa-arrow-circle-right"),
			pango.Icon("fa-cloud"),
			pango.Icon("fa-music"),
			pango.Icon("fa-tags"),
		)
		require.NotEmpty(t, testIcons.String(), "No expected solid icons were loaded")

		testIcons = pango.New(
			pango.Icon("far-bell"),
			pango.Icon("far-compass"),
			pango.Icon("far-paper-plane"),
			pango.Icon("far-user-circle"),
		)
		require.NotEmpty(t, testIcons.String(), "No expected regular icons were loaded")

		testIcons = pango.New(
			pango.Icon("fab-android"),
			pango.Icon("fab-css3-alt"),
			pango.Icon("fab-empire"),
			pango.Icon("fab-linux"),
		)
		require.NotEmpty(t, testIcons.String(), "No expected brand icons were loaded")
		return nil
	})
}
