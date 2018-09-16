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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"barista.run/testing/mockio"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const configFile = "/conf/dir/127.0.0.1_SK3VYxHdcs7MqflZ6QeO0YmAS0jCBLwdq73pqA.json"

var testEndpoint oauth2.Endpoint
var checkUrl string
var tokenExpirySeconds = 120

func TestConfigDir(t *testing.T) {
	os.Setenv("XDG_CONFIG_HOME", "/xdg/config/")
	require.Equal(t, os.ExpandEnv("/xdg/config/barista/oauth"), getConfigDir())
	os.Unsetenv("XDG_CONFIG_HOME")
	require.Equal(t, os.ExpandEnv("$HOME/.config/barista/oauth"), getConfigDir())
}

func TestNoOauthSetup(t *testing.T) {
	defer func(args []string) { os.Args = args }(os.Args)

	mockStdin := mockio.Stdin()
	stdin = mockStdin
	mockStdout := mockio.Stdout()
	stdout = mockStdout
	exitCode := make(chan int, 1)
	osExit = func(code int) { exitCode <- code }
	resetKey([]byte("test"))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	require.Equal(t, 0, <-exitCode, "exits even if no tokens are registered")
	out := mockStdout.ReadNow()
	require.Equal(t, "Nothing to update\n", out,
		"simple setup process when no tokens are registered")

	os.Args = []string{}
	InteractiveSetup()
	select {
	case <-exitCode:
		require.Fail(t, "os.Exit called in non-oauth setup mode")
	default:
	}
	require.Empty(t, mockStdout.ReadNow(),
		"no writes when not running oauth setup")

	os.Args = []string{"foo", "bar"}
	InteractiveSetup()
	select {
	case <-exitCode:
		require.Fail(t, "os.Exit called in non-oauth setup mode")
	default:
	}
	require.Empty(t, mockStdout.ReadNow(),
		"no writes when not running oauth setup")
}

func assertExitCode(t *testing.T, exitCode <-chan int, expected int) {
	select {
	case code := <-exitCode:
		require.Equal(t, expected, code)
	case <-time.After(time.Second):
		require.Fail(t, "OAuth setup did not exit")
	}
}

func resetForTest() (mockStdout *mockio.Writable, mockStdin *mockio.Readable, exitCode chan int) {
	registeredConfigsMu.Lock()
	defer registeredConfigsMu.Unlock()
	registeredConfigs = nil
	fs = afero.NewMemMapFs()
	randRead = successfulRandRead
	mockStdin = mockio.Stdin()
	stdin = mockStdin
	mockStdout = mockio.Stdout()
	stdout = mockStdout
	configDir = "/conf/dir"
	exitCode = make(chan int, 1)
	osExit = func(code int) { exitCode <- code }
	resetKey([]byte("test"))
	return // mockStdout, mockStdin, exitCode
}

func TestOauthSetup(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()

	conf := Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	exists, _ := afero.Exists(fs, configFile)
	require.False(exists, "file not created on registration")

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	out, _ := mockStdout.ReadUntil('>', time.Second)
	require.Regexp(
		`Updating registered Oauth configurations:

\[1 of 1\] barista.run/oauth.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: a, b
- Visit http://127.0.0.1:\d+/auth\?.*? and enter the code here:
>`, out)

	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 0)
	require.Regexp(
		`\+ Successfully updated token, expires .*?

All tokens updated successfully
`, mockStdout.ReadNow())

	exists, _ = afero.Exists(fs, configFile)
	require.True(exists, "file created on success")

	client := conf.Client()
	resp, _ := client.Get(checkUrl)
	require.Equal(200, resp.StatusCode)
}

func TestOauthMultiSetup(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()

	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "OtherClientID",
		ClientSecret: "also-not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"c"},
	})

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	mockStdin.Write([]byte("authcode\n"))
	mockStdin.Write([]byte("authcode\n"))
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 0)

	require.Regexp(
		`Updating registered Oauth configurations:

\[1 of 3\] barista.run/oauth.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: a, b
- Visit http://127.0.0.1:\d+/auth\?.*? and enter the code here:
> \+ Successfully updated token, expires .*?

\[2 of 3\] barista.run/oauth.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: a, b
- Visit http://127.0.0.1:\d+/auth\?.*? and enter the code here:
> \+ Successfully updated token, expires .*?

\[3 of 3\] barista.run/oauth.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: c
- Visit http://127.0.0.1:\d+/auth\?.*? and enter the code here:
> \+ Successfully updated token, expires .*?

All tokens updated successfully
`, mockStdout.ReadNow())

	entries, _ := afero.ReadDir(fs, "/conf/dir/")
	require.Equal(3, len(entries), "All tokens saved to separate files")
}

func TestOauthSavedToken(t *testing.T) {
	require := require.New(t)
	mockStdout, _, exitCode := resetForTest()

	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	require.NoError(storeToken(configFile,
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	assertExitCode(t, exitCode, 0)
	require.Regexp(
		`Updating registered Oauth configurations:

\[1 of 1\] barista\.run/oauth\.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: a, b
\+ Using saved token, never expires

All tokens updated successfully
`, mockStdout.ReadNow())
}

func TestOauthExpiredToken(t *testing.T) {
	require := require.New(t)
	mockStdout, _, exitCode := resetForTest()

	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	require.NoError(storeToken(configFile,
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
			Expiry:       time.Now().Add(-time.Hour),
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	assertExitCode(t, exitCode, 0)
	require.Regexp(
		`Updating registered Oauth configurations:

\[1 of 1\] barista\.run/oauth\.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: a, b
\+ Attempting automatic token refresh
\+ Using saved token, expires .*?

All tokens updated successfully
`, mockStdout.ReadNow())
}

func TestOauthInvalidSavedToken(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()

	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	require.NoError(storeToken(configFile,
		&oauth2.Token{
			AccessToken:  "not-mocktoken",
			RefreshToken: "not-mockrefreshtoken",
			Expiry:       time.Now().Add(-time.Hour),
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	out, _ := mockStdout.ReadUntil('>', time.Second)
	require.Regexp(
		`Updating registered Oauth configurations:

\[1 of 1\] barista.run/oauth.\w+
\* Domain: 127\.0\.0\.1
\* Scopes: a, b
\+ Attempting automatic token refresh
! Automatic refresh failed
- Visit http://127.0.0.1:\d+/auth\?.*? and enter the code here:
>`, out)

	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 0)
	require.Regexp(
		`\+ Successfully updated token, expires .*?

All tokens updated successfully
`, mockStdout.ReadNow())
}

func TestOauthSetupStdinError(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()
	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})

	os.Args = []string{"arg0", "setup-oauth"}
	mockStdin.ShouldError(errors.New("stdin error"))
	go InteractiveSetup()
	mockStdout.ReadUntil('>', time.Second)
	mockStdout.ReadUntil(' ', time.Second)

	assertExitCode(t, exitCode, 1)
	out := mockStdout.ReadNow()
	require.Equal("! Failed to update token: stdin error\n", out)
}

func TestOauthSetupEndpointError(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()
	Register(&oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  testEndpoint.AuthURL + "/auth-404",
			TokenURL: testEndpoint.TokenURL + "/token-404",
		},
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})

	go InteractiveSetup()
	mockStdout.ReadUntil('>', time.Second)
	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 1)
	require.Regexp(`! Failed to update token: .*`, mockStdout.ReadNow())
}

func TestOauthSetupInvalidCode(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()
	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})

	go InteractiveSetup()
	mockStdout.ReadUntil('>', time.Second)
	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("not-authcode\n"))
	assertExitCode(t, exitCode, 1)
	require.Regexp(`! Failed to update token: .*`, mockStdout.ReadNow())
}

func TestOauthSetupRandError(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()
	randRead = errorRandRead
	Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})

	go InteractiveSetup()
	mockStdout.ReadUntil('>', time.Second)
	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 1)
	require.Regexp(`! Failed to update token: .*`, mockStdout.ReadNow())
}

func TestOauthLoadSavedToken(t *testing.T) {
	require := require.New(t)
	resetForTest()

	conf := Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	require.NoError(storeToken(configFile,
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
		}))

	client := conf.Client()
	resp, _ := client.Get(checkUrl)
	require.Equal(200, resp.StatusCode)
}

func TestOauthLoadNoSavedToken(t *testing.T) {
	require := require.New(t)
	resetForTest()

	conf := Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})

	client := conf.Client()
	_, err := client.Get(checkUrl)
	require.Error(err, "when no token is available")
}

func TestOauthTokenAutoRefresh(t *testing.T) {
	require := require.New(t)
	resetForTest()

	conf := Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	require.NoError(storeToken(configFile,
		&oauth2.Token{
			AccessToken:  "whatever",
			RefreshToken: "mockrefreshtoken",
			Expiry:       time.Now().Add(-time.Hour),
		}))

	client := conf.Client()
	resp, _ := client.Get(checkUrl)
	require.Equal(200, resp.StatusCode)

	tok, err := loadToken(configFile)
	require.NoError(err)
	require.Equal("mocktoken", tok.AccessToken, "updated token written to file")
}

func TestOauthLoadInvalidSavedToken(t *testing.T) {
	require := require.New(t)
	resetForTest()

	conf := Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
	require.NoError(storeToken(configFile,
		&oauth2.Token{
			AccessToken:  "not-mocktoken",
			RefreshToken: "not-mockrefreshtoken",
		}))

	client := conf.Client()
	resp, _ := client.Get(checkUrl)
	require.Equal(401, resp.StatusCode)
}

func TestMain(m *testing.M) {
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("code") == "authcode" {
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
			w.Write([]byte(strings.Join([]string{
				"access_token=mocktoken",
				"token_type=bearer",
				fmt.Sprintf("expires_in=%d", tokenExpirySeconds),
				"refresh_token=mockrefreshtoken",
			}, "&")))
			return
		}
		if r.FormValue("refresh_token") == "mockrefreshtoken" {
			w.Header().Set("Content-Type", "application/x-www-form-urlencoded")
			w.Write([]byte(strings.Join([]string{
				"access_token=mocktoken",
				"token_type=bearer",
				fmt.Sprintf("expires_in=%d", tokenExpirySeconds),
			}, "&")))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mocktoken" {
			http.Error(w, "Missing Auth Header", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	testEndpoint = oauth2.Endpoint{
		AuthURL:  server.URL + "/auth",
		TokenURL: server.URL + "/token",
	}
	checkUrl = server.URL + "/check"

	os.Exit(m.Run())
}
