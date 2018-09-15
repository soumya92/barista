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

package oauth

import (
	"errors"
	"io/ioutil"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func resetKey(key []byte) {
	globalEncryptionKeyMu.Lock()
	globalEncryptionKey = key
	globalEncryptionKeyMu.Unlock()
}

func successfulRandRead(in []byte) (n int, e error) {
	for i := 0; i < len(in); i++ {
		in[i] = byte(i % 256)
	}
	return len(in), nil
}

func errorRandRead(in []byte) (n int, e error) {
	return 0, errors.New("something went wrong")
}

func errorAfterFirstAttempt() func([]byte) (int, error) {
	passed := false
	return func(in []byte) (int, error) {
		if passed {
			return errorRandRead(in)
		}
		passed = true
		return successfulRandRead(in)
	}
}

func TestEncryptionKey(t *testing.T) {
	resetKey(nil)
	defer resetKey(nil)

	require.Panics(t, func() { getEncryptionKeyChecked() }, "with no key set")

	SetEncryptionKey(nil)
	require.Panics(t, func() { getEncryptionKeyChecked() }, "with nil key set")

	SetEncryptionKey([]byte("foobar"))
	require.NotPanics(t, func() { getEncryptionKeyChecked() }, "with key set")
	require.Panics(t, func() { SetEncryptionKey([]byte("new")) }, "with key set")
}

func TestCryptStore(t *testing.T) {
	resetKey([]byte("abcd"))
	defer resetKey(nil)

	fs = afero.NewMemMapFs()
	randRead = successfulRandRead

	require.NoError(t, storeToken("empty.json", &oauth2.Token{}))
	savedFile, _ := afero.ReadFile(fs, "empty.json")
	expectedFile, _ := ioutil.ReadFile("testdata/empty.json")
	require.Equal(t, expectedFile, savedFile)

	require.NoError(t, storeToken("simple.json", &oauth2.Token{
		AccessToken:  "foobar",
		RefreshToken: "supersecret",
		// TODO: Investigate why this fails on CI.
		// Expiry:       time.Unix(1500000000, 0),
	}))
	savedFile, _ = afero.ReadFile(fs, "simple.json")
	expectedFile, _ = ioutil.ReadFile("testdata/simple.json")
	require.Equal(t, expectedFile, savedFile)

	loaded, err := loadToken("empty.json")
	require.NoError(t, err)
	require.Equal(t, oauth2.Token{}, *loaded)

	loaded, err = loadToken("simple.json")
	require.Equal(t, oauth2.Token{
		AccessToken:  "foobar",
		RefreshToken: "supersecret",
	}, *loaded)

	_, err = loadToken("nonexistent.json")
	require.Error(t, err)

	resetKey([]byte("not it"))
	_, err = loadToken("empty.json")
	require.Error(t, err, "with wrong key")

	afero.WriteFile(fs, "not-json.txt", []byte(`not-json`), 0644)
	_, err = loadToken("not-json.txt")
	require.Error(t, err)

	randRead = errorRandRead
	require.Error(t, storeToken("foo.json", &oauth2.Token{}))
	exists, _ := afero.Exists(fs, "foo.json")
	require.False(t, exists, "file not written on error")

	randRead = errorAfterFirstAttempt()
	require.Error(t, storeToken("foo.json", &oauth2.Token{}))
	exists, _ = afero.Exists(fs, "foo.json")
	require.False(t, exists, "file not written on error")
}
