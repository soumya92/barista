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
	"net/url"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"barista.run/testing/mockio"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func testHostname() string {
	u, _ := url.Parse(checkURL)
	return u.Hostname()
}

func configFile() string {
	return fmt.Sprintf("/conf/dir/%s_SK3VYxHdcs7MqflZ6QeO0YmAS0jCBLwdq73pqA.json", testHostname())
}

var testEndpoint oauth2.Endpoint
var checkURL string
var tokenExpirySeconds = 120

func TestConfigDir(t *testing.T) {
	os.Setenv("XDG_CONFIG_HOME", "/xdg/config/")
	require.Equal(t, os.ExpandEnv("/xdg/config/barista/oauth"), getConfigDir())
	os.Unsetenv("XDG_CONFIG_HOME")
	require.Equal(t, os.ExpandEnv("$HOME/.config/barista/oauth"), getConfigDir())
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
	setupHasBeenCalled = 0
	registeredConfigs = nil
	registeredConfigsMap = map[string]*Config{}
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

func myPackage() string {
	pc, _, _, _ := runtime.Caller(0)
	return strings.TrimSuffix(runtime.FuncForPC(pc).Name(), ".myPackage")
}

var pkgName = myPackage()
var testMethod = regexp.MustCompile(`\.Test\w+`)
var authURL = regexp.MustCompile(`http://(127\.\d+\.\d+\.\d+|\[::1\])\:\d+\/auth\?[^ ]+`)
var expiry = regexp.MustCompile(`\d{1,2} \w{3} \d{1,2} \d{1,2}:\d{2} \w+`)

// sanitiseOauthOutput replaces some dynamic values with placeholders to allow
// equality assertions in tests. Using equality in tests instead of regexes has
// two advantages:
// - no need to escape the expected output.
// - nice diff view when the test fails.
func sanitiseOauthOutput(str string) string {
	str = testMethod.ReplaceAllString(str, ".#testName#")
	str = authURL.ReplaceAllString(str, "#authURL#")
	str = expiry.ReplaceAllString(str, "#expiry#")
	str = strings.Replace(str, pkgName, "#pkg#", -1)
	str = strings.Replace(str, testHostname(), "#host#", -1)
	return str
}

func TestNoOauthSetup(t *testing.T) {
	defer func(args []string) { os.Args = args }(os.Args)

	mockStdout, _, exitCode := resetForTest()
	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	require.Equal(t, 0, <-exitCode, "exits even if no tokens are registered")
	out := mockStdout.ReadNow()
	require.Equal(t, "Nothing to update\n", out,
		"simple setup process when no tokens are registered")

	mockStdout, _, exitCode = resetForTest()
	os.Args = []string{}
	InteractiveSetup()
	select {
	case <-exitCode:
		require.Fail(t, "os.Exit called in non-oauth setup mode")
	default:
	}
	require.Empty(t, mockStdout.ReadNow(),
		"no writes when not running oauth setup")

	mockStdout, _, exitCode = resetForTest()
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
	exists, _ := afero.Exists(fs, configFile())
	require.False(exists, "file not created on registration")

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	out, _ := mockStdout.ReadUntil('>', time.Second)
	require.Equal(
		`Updating registered Oauth configurations:

[1 of 1] #pkg#.#testName#
* Domain: #host#
* Scopes: a, b
- Visit #authURL# and enter the code here:
>`, sanitiseOauthOutput(out))

	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 0)
	require.Equal(
		`+ Successfully updated token, expires #expiry#

All tokens updated successfully
`, sanitiseOauthOutput(mockStdout.ReadNow()))

	exists, _ = afero.Exists(fs, configFile())
	require.True(exists, "file created on success")

	client, err := conf.Client()
	require.NoError(err)
	resp, _ := client.Get(checkURL)
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

	require.Equal(
		`Updating registered Oauth configurations:

[1 of 3] #pkg#.#testName#
* Domain: #host#
* Scopes: a, b
- Visit #authURL# and enter the code here:
> + Successfully updated token, expires #expiry#

[2 of 3] #pkg#.#testName#
* Domain: #host#
* Scopes: a, b
- Visit #authURL# and enter the code here:
> + Successfully updated token, expires #expiry#

[3 of 3] #pkg#.#testName#
* Domain: #host#
* Scopes: c
- Visit #authURL# and enter the code here:
> + Successfully updated token, expires #expiry#

All tokens updated successfully
`, sanitiseOauthOutput(mockStdout.ReadNow()))

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
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	assertExitCode(t, exitCode, 0)
	require.Equal(
		`Updating registered Oauth configurations:

[1 of 1] #pkg#.#testName#
* Domain: #host#
* Scopes: a, b
+ Using saved token, never expires

All tokens updated successfully
`, sanitiseOauthOutput(mockStdout.ReadNow()))
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
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
			Expiry:       time.Now().Add(-time.Hour),
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	assertExitCode(t, exitCode, 0)
	require.Equal(
		`Updating registered Oauth configurations:

[1 of 1] #pkg#.#testName#
* Domain: #host#
* Scopes: a, b
+ Attempting automatic token refresh
+ Using saved token, expires #expiry#

All tokens updated successfully
`, sanitiseOauthOutput(mockStdout.ReadNow()))
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
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "not-mocktoken",
			RefreshToken: "not-mockrefreshtoken",
			Expiry:       time.Now().Add(-time.Hour),
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	out, _ := mockStdout.ReadUntil('>', time.Second)
	require.Equal(
		`Updating registered Oauth configurations:

[1 of 1] #pkg#.#testName#
* Domain: #host#
* Scopes: a, b
+ Attempting automatic token refresh
! Automatic refresh failed
- Visit #authURL# and enter the code here:
>`, sanitiseOauthOutput(out))

	mockStdout.ReadUntil(' ', time.Second)
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 0)
	require.Equal(
		`+ Successfully updated token, expires #expiry#

All tokens updated successfully
`, sanitiseOauthOutput(mockStdout.ReadNow()))
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
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
		}))

	client, err := conf.Client()
	require.NoError(err)
	resp, _ := client.Get(checkURL)
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

	client, err := conf.Client()
	require.Error(err, "when no token is available")
	require.NotNil(client, "Still returns a client")
	_, err = client.Get(checkURL)
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
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "whatever",
			RefreshToken: "mockrefreshtoken",
			Expiry:       time.Now().Add(-time.Hour),
		}))

	client, err := conf.Client()
	require.NoError(err)
	resp, _ := client.Get(checkURL)
	require.Equal(200, resp.StatusCode)

	tok, err := loadToken(configFile())
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
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "not-mocktoken",
			RefreshToken: "not-mockrefreshtoken",
		}))

	client, err := conf.Client()
	require.NoError(err, "Even if stored token is invalid")
	resp, _ := client.Get(checkURL)
	require.Equal(401, resp.StatusCode)
}

func registerA() *Config {
	return Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
}

func registerB() *Config {
	return Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"a", "b"},
	})
}

func registerC() *Config {
	return Register(&oauth2.Config{
		Endpoint:     testEndpoint,
		ClientID:     "ClientID",
		ClientSecret: "not-really-secret",
		RedirectURL:  "localhost:1",
		Scopes:       []string{"c"},
	})
}

func TestOauthConfigReuse(t *testing.T) {
	require := require.New(t)
	mockStdout, mockStdin, exitCode := resetForTest()

	confA := registerA()
	confA2 := registerA()
	confB := registerB()
	confC := registerC()

	require.NotEqual(confA, confC, "different scopes")
	require.True(confA == confA2, "re-uses config")
	require.True(confA == confB, "re-uses config across methods")

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	mockStdin.Write([]byte("authcode\n"))
	mockStdin.Write([]byte("authcode\n"))
	assertExitCode(t, exitCode, 0)

	require.Equal(
		`Updating registered Oauth configurations:

[1 of 2] #pkg#.registerA, #pkg#.registerB
* Domain: #host#
* Scopes: a, b
- Visit #authURL# and enter the code here:
> + Successfully updated token, expires #expiry#

[2 of 2] #pkg#.registerC
* Domain: #host#
* Scopes: c
- Visit #authURL# and enter the code here:
> + Successfully updated token, expires #expiry#

All tokens updated successfully
`, sanitiseOauthOutput(mockStdout.ReadNow()))
}

func TestOauthRegisterAfterSetup(t *testing.T) {
	require := require.New(t)
	mockStdout, _, exitCode := resetForTest()

	require.NotPanics(func() { registerA() }, "Register before setup")
	require.NoError(storeToken(configFile(),
		&oauth2.Token{
			AccessToken:  "mocktoken",
			RefreshToken: "mockrefreshtoken",
		}))

	os.Args = []string{"arg0", "setup-oauth"}
	go InteractiveSetup()
	assertExitCode(t, exitCode, 0)
	mockStdout.ReadNow() // empty the buffer.

	require.Panics(func() { registerB() }, "Register after setup")
	InteractiveSetup() // should be nop, so main thread is fine.
	select {
	case <-exitCode:
		require.Fail("Should not call os.Exit on second call to setup")
	default:
	}
	require.Empty(mockStdout.ReadNow(), "No output on second call to setup")
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
	checkURL = server.URL + "/check"

	os.Exit(m.Run())
}
