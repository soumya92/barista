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

package mdi

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

	afero.WriteFile(fs, "/src/material-error-1/scss/_variables.scss", []byte(
		`
$blah: "red";
$foobar: "green";

$mdi-icons: (
	-- error --
    "someIcon": 61,
    "otherIcon": 62,
    "badIcon": food
);`,
	), 0644)
	require.Error(t, Load("/src/material-error-1"))

	afero.WriteFile(fs, "/src/material-error-2/scss/_variables.scss", nil, 0644)
	require.Error(t, Load("/src/material-error-2"))

	afero.WriteFile(fs, "/src/material-error-3/scss/_variables.scss", []byte(
		`
$blah: "red";
$foobar: "green";

$mdi-icons: (
    "someIcon": 61,
    "otherIcon": 62,
    "badIcon": food
);

$randomStuff: "yellow";`,
	), 0644)
	require.Error(t, Load("/src/material-error-3"))

	afero.WriteFile(fs, "/src/material-error-4/scss/_variables.scss", []byte(
		`
$blah: "red";
$foobar: "green";

$mdi-icons: (
    "someIcon": 61,
`,
	), 0644)
	require.Error(t, Load("/src/material-error-4"))
}

func TestValid(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/material/scss/_variables.scss", []byte(
		`
$blah: "red";
$foobar: "green";

$mdi-icons: (
    "someIcon": 61,
    "otherIcon": 62
);`,
	), 0644)
	require.NoError(t, Load("/src/material"))
	pangoTesting.AssertText(t, "a", pango.Icon("mdi-someIcon").String())
}

// TestLive tests that current master branch of the icon font works with
// this package. This test only runs when CI runs tests in 'cron' mode,
// which provides timely notifications of incompatible changes while
// keeping default tests hermetic.
func TestLive(t *testing.T) {
	fs = githubfs.New()
	cron.Test(t, func() error {
		if err := Load("/Templarian/MaterialDesign-Webfont/master"); err != nil {
			return err
		}
		// At least one of these icons should be loaded.
		testIcons := pango.New(
			pango.Icon("mdi-arrow-right-thick"),
			pango.Icon("mdi-bullhorn"),
			pango.Icon("mdi-close-octagon"),
			pango.Icon("mdi-scale-balance"),
		)
		require.NotEmpty(t, testIcons.String(), "No expected icons were loaded")
		return nil
	})
}
