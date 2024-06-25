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

package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soumya92/barista/testing/httpserver"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestWrapper(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
	defer okServer.Close()

	testServer := httpserver.New()
	defer testServer.Close()
	url404 := testServer.URL + "/code/404"
	redirLoop := testServer.URL + "/redir"

	client := testServer.Client()
	r, err := client.Get(url404)
	require.NoError(t, err)
	require.Equal(t, 404, r.StatusCode)

	_, err = client.Get(redirLoop)
	require.Error(t, err)

	Wrap(client, okServer.URL)
	r, err = client.Get(url404)
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)

	_, err = client.Get(redirLoop)
	require.NoError(t, err)

	r, err = client.Get("https://test.example.org/not-found")
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)

	req, err := http.NewRequest("GET", "https://api.example.net/endpoint", nil)
	require.NoError(t, err)
	require.Equal(t, "https://api.example.net/endpoint", req.URL.String())
	r, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)
	require.Equal(t, "https://api.example.net/endpoint", req.URL.String())
}

func TestOauthTokenFreezing(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer mocktoken" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
	defer tokenServer.Close()
	var someURL = tokenServer.URL + "/foo"

	client := tokenServer.Client()
	r, err := client.Get(someURL)
	require.NoError(t, err)
	require.Equal(t, 403, r.StatusCode)

	FreezeOauthToken(client, "mocktoken")
	r, err = client.Get(someURL)
	require.NoError(t, err)
	require.Equal(t, 403, r.StatusCode)

	oauthClient := (&oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  tokenServer.URL + "/auth",
			TokenURL: tokenServer.URL + "/token",
		},
		ClientID:     "whatever",
		ClientSecret: "not-really-secret",
	}).Client(context.Background(), nil)

	_, err = oauthClient.Get(someURL)
	require.Error(t, err)

	FreezeOauthToken(oauthClient, "mocktoken")
	r, err = oauthClient.Get(someURL)
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)

	oauthClient = (&oauth2.Config{
		Endpoint: oauth2.Endpoint{
			AuthURL:  tokenServer.URL + "/auth",
			TokenURL: tokenServer.URL + "/token",
		},
		ClientID:     "whatever",
		ClientSecret: "not-really-secret",
	}).Client(context.Background(), &oauth2.Token{
		AccessToken: "wrongtoken",
	})

	r, err = oauthClient.Get(someURL)
	require.NoError(t, err)
	require.Equal(t, 403, r.StatusCode)

	FreezeOauthToken(oauthClient, "mocktoken")
	r, err = oauthClient.Get(someURL)
	require.NoError(t, err)
	require.Equal(t, 200, r.StatusCode)
}
