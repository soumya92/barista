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

// Package oauth provides oauth capabilities to barista and modules.
package oauth // import "barista.run/oauth"

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	l "barista.run/logging"

	"golang.org/x/oauth2"
)

// Config represents a oauth configuration for use in barista.
// It can be used to create an authenticated client after the user has setup
// oauth for barista using the InteractiveSetup() method.
type Config struct {
	config   *oauth2.Config
	filename string
	// For more context during interactive auth
	domain  string
	callers []string
	// To support automatic saving of refreshed tokens.
	tokenSource oauth2.TokenSource
	token       *oauth2.Token
	mu          sync.Mutex
}

// Track all registered configs, so that InteractiveSetup() can work.
var (
	registeredConfigs    = []*Config{}
	registeredConfigsMap = map[string]*Config{}
	registeredConfigsMu  sync.Mutex
)
var configDir = getConfigDir()

// Since any tokens registered *after* setup has been called will not work,
// we'll track when setup is called and panic on registrations after.
var setupHasBeenCalled int32 // atomic bool

// getConfigDir gets an XDG compliant directory for storing encrypted tokens.
func getConfigDir() string {
	configRoot := os.ExpandEnv("$HOME/.config")
	if xdgConfig, ok := os.LookupEnv("XDG_CONFIG_HOME"); ok {
		configRoot = xdgConfig
	}
	return filepath.Join(configRoot, "barista", "oauth")
}

// Register registers an oauth2 configuration with barista's oauth package.
// Only configurations that are registered *before* Run() is called will be
// added to the interactive oauth setup, so modules should usually call this
// either in init() or in their New() functions.
func Register(config *oauth2.Config) *Config {
	if atomic.LoadInt32(&setupHasBeenCalled) != 0 {
		panic("Cannot register after setup has been called!")
	}
	providerU, _ := url.Parse(config.Endpoint.AuthURL)
	c := &Config{
		config: config,
		domain: providerU.Hostname(),
	}
	caller := "<unknown>"
	pc, _, _, ok := runtime.Caller(1)
	if ok {
		caller = runtime.FuncForPC(pc).Name()
	}
	hasher := sha256.New224()
	// Each token will be stored in config dir, in the form $provider_$hash.
	// $provider comes from the domain, to allow users to see which providers
	// have been used, and backup/remove specific providers.
	// $hash comes from hashing the scopes and client-id, to allow multiple
	// modules to use the same provider with their own configuration.
	io.WriteString(hasher, config.ClientID)
	for _, scope := range config.Scopes {
		io.WriteString(hasher, scope)
	}
	hash := base64.RawURLEncoding.EncodeToString(hasher.Sum(nil))
	filename := filepath.Join(configDir,
		fmt.Sprintf("%s_%s.json", c.domain, hash))
	// So the final resulting filename will be something like
	// ~/.config/barista/oauth/accounts.google.com_MTIzNDU2Nzg5MDEyMzQ1Njc4OTAxMjM0NTY3OA.json
	c.filename = filename
	registeredConfigsMu.Lock()
	defer registeredConfigsMu.Unlock()
	existing, ok := registeredConfigsMap[filename]
	if ok {
		existing.addCaller(caller)
		return existing
	}
	c.callers = []string{caller}
	registeredConfigs = append(registeredConfigs, c)
	registeredConfigsMap[filename] = c
	return c
}

func (c *Config) addCaller(caller string) {
	for _, cr := range c.callers {
		if cr == caller {
			return
		}
	}
	c.callers = append(c.callers, caller)
}

// for tests.
var stdin io.Reader = os.Stdin
var stdout io.Writer = os.Stdout
var osExit = os.Exit

// InteractiveSetup checks each registered config and guides the user through
// a command-line interactive auth process for each config that doesn't have a
// valid token. Only intended for use by barista's Run() method, calling it
// at the wrong time can leave you unable to save tokens for some configs.
func InteractiveSetup() {
	if !atomic.CompareAndSwapInt32(&setupHasBeenCalled, 0, 1) {
		l.Log("Setup called more than once!")
		return
	}
	if len(os.Args) < 2 || os.Args[1] != "setup-oauth" {
		return
	}
	// TODO: Argument parsing here.
	forceUpdate := len(os.Args) > 2 && os.Args[2] == "force-refresh"
	registeredConfigsMu.Lock()
	defer registeredConfigsMu.Unlock()
	if len(registeredConfigs) == 0 {
		fmt.Fprintln(stdout, "Nothing to update")
		osExit(0)
		return
	}
	os.MkdirAll(configDir, 0700)
	success := true
	fmt.Fprintln(stdout, "Updating registered Oauth configurations:")
	for idx, c := range registeredConfigs {
		if !c.prompt(idx, len(registeredConfigs), forceUpdate) {
			success = false
		}
	}
	if !success {
		osExit(1)
		return
	}
	fmt.Fprintln(stdout, "\nAll tokens updated successfully")
	osExit(0)
}

func commas(s []string) string {
	return strings.Join(s, ", ")
}

func (c *Config) prompt(index, total int, force bool) bool {
	fmt.Fprintf(stdout, "\n[%d of %d] %s\n* Domain: %s\n* Scopes: %s\n",
		index+1, total, commas(c.callers), c.domain, commas(c.config.Scopes))

	err := c.autoUpdateToken()
	if err == nil {
		if force && c.token.RefreshToken != "" {
			c.tokenSource = c.config.TokenSource(context.Background(), c.token)
			c.token.Expiry = time.Now().Add(-time.Hour)
		}
		if !c.token.Valid() {
			fmt.Fprintf(stdout, "+ Attempting automatic token refresh\n")
			_, err = c.Token()
		}
		if err == nil {
			// TODO: Figure out how to check the token for server-side validity.
			// Even if the time is not expired, the token could be invalidated
			// through other means, e.g password change or access revocation.
			// Currently the only solution is for the user to delete the json.
			fmt.Fprintf(stdout, "+ Using saved token, %s\n",
				formatExpiry(c.token.Expiry))
			return true
		}
		fmt.Fprintf(stdout, "! Automatic refresh failed\n")
	}

	authURL := c.config.AuthCodeURL("no-state", oauth2.AccessTypeOffline)
	fmt.Fprintf(stdout, "- Visit %v and enter the code here:\n> ", authURL)
	var authCode string
	if _, err = fmt.Fscan(stdin, &authCode); err == nil {
		c.token, err = c.config.Exchange(oauth2.NoContext, authCode)
	}
	if err == nil {
		err = storeToken(c.filename, c.token)
	}
	if err != nil {
		fmt.Fprintf(stdout, "! Failed to update token: %v\n", err)
		return false
	}
	fmt.Fprintf(stdout, "+ Successfully updated token, %s\n",
		formatExpiry(c.token.Expiry))
	return true
}

func formatExpiry(expiry time.Time) string {
	if expiry.IsZero() {
		return "never expires"
	}
	return fmt.Sprintf("expires %s", expiry.Format(time.RFC822))
}

func (c *Config) autoUpdateToken() error {
	var err error
	c.token, err = loadToken(c.filename)
	return err
}

// Token makes Config a TokenSource, in a way that automatically saves any
// newly fetched tokens to disk.
func (c *Config) Token() (*oauth2.Token, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tokenSource == nil {
		if err := c.autoUpdateToken(); err != nil {
			return nil, err
		}
		c.tokenSource = c.config.TokenSource(context.Background(), c.token)
	}
	if c.token.Valid() {
		return c.token, nil
	}
	tok, err := c.tokenSource.Token()
	if err != nil {
		return nil, err
	}
	c.token = tok
	return tok, storeToken(c.filename, tok)
}

// Client returns an http client that authorises requests using the previously
// saved token for this configuration.
func (c *Config) Client() (*http.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return oauth2.NewClient(context.Background(), c), c.autoUpdateToken()
}
