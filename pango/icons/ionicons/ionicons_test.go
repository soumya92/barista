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

package ionicons

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

	afero.WriteFile(fs, "/src/ion-error-1/scripts/manifest.json", []byte(
		`-- Invalid JSON --`,
	), 0644)
	require.Error(t, LoadIos("/src/ion-error-1"))

	afero.WriteFile(fs, "/src/ion-error-2/scripts/manifest.json", nil, 0644)
	require.Error(t, LoadMd("/src/ion-error-2"))

	afero.WriteFile(fs, "/src/ion-error-3/scripts/manifest.json", []byte(
		`{"icons": [
			{"name": "someIcon", "code": "0x61"},
			{"name": "otherIcon", "code": "0x62"},
			{"name": "someIcon", "code": "0xghij"}
		]}`,
	), 0644)
	require.Error(t, Load("/src/ion-error-3"))
}

func TestValid(t *testing.T) {
	fs = afero.NewMemMapFs()
	afero.WriteFile(fs, "/src/ion/scripts/manifest.json", []byte(
		`{"icons": [
			{"name": "md-someIcon", "code": "0x61"},
			{"name": "ios-someIcon", "code": "0x62"},
			{"name": "otherIcon", "code": "0x63"}
		]}`,
	), 0644)
	require.NoError(t, Load("/src/ion"))
	pangoTesting.AssertText(t, "a", pango.Icon("ion-md-someIcon").String())
	pangoTesting.AssertText(t, "b", pango.Icon("ion-ios-someIcon").String())
	pangoTesting.AssertText(t, "c", pango.Icon("ion-otherIcon").String())

	require.NoError(t, LoadMd("/src/ion"))
	pangoTesting.AssertText(t, "a", pango.Icon("ion-someIcon").String())
	pangoTesting.AssertText(t, "b", pango.Icon("ion-ios-someIcon").String())

	require.NoError(t, LoadIos("/src/ion"))
	pangoTesting.AssertText(t, "b", pango.Icon("ion-someIcon").String())
	pangoTesting.AssertText(t, "a", pango.Icon("ion-md-someIcon").String())
}

// TestLive tests that current master branch of the icon font works with
// this package. This test only runs when CI runs tests in 'cron' mode,
// which provides timely notifications of incompatible changes while
// keeping default tests hermetic.
func TestLive(t *testing.T) {
	fs = githubfs.New()
	cron.Test(t, func() error {
		if err := Load("/ionic-team/ionicons/master"); err != nil {
			return err
		}
		// At least one of these icons should be loaded.
		testIcons := pango.New(
			pango.Icon("ion-md-book"),
			pango.Icon("ion-ios-help"),
			pango.Icon("ion-md-headset"),
			pango.Icon("ion-ios-trash"),
		)
		require.NotEmpty(t, testIcons.String(), "No expected icons were loaded")
		return nil
	})
}
