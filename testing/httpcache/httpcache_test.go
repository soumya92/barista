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

package httpcache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"

	testServer "barista.run/testing/httpserver"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	server := testServer.New()
	defer server.Close()
	u, _ := url.Parse(server.URL)
	base := sanitizeRe.ReplaceAllString(u.Host, "-")

	dir, err := ioutil.TempDir("", "httpcache")
	if err != nil {
		t.Fatalf("failed to create test directory: %s", err)
	}
	defer os.RemoveAll(dir)

	os.Setenv("XDG_CACHE_HOME", dir)
	client := &http.Client{Transport: Wrap(http.DefaultTransport)}

	client.Get(server.URL + "/basic/foo")
	_, err = os.Stat(fmt.Sprintf("%s/barista/http/%s_basic-foo", dir, base))
	require.NoError(t, err, "simple response cached to correct file")

	client.Get(server.URL + "/code/404")
	_, err = os.Stat(fmt.Sprintf("%s/barista/http/%s_code-404", dir, base))
	require.NoError(t, err, "http non-200 responses also cached")

	_, redirectError := client.Get(server.URL + "/redir")

	r, err := client.Get(server.URL + "/tpl/debug?param=foo&bar=baz")
	require.NoError(t, err)
	body, _ := ioutil.ReadAll(r.Body)
	require.Equal(t, "\nbar = baz\nparam = foo\n", string(body))

	server.Close()

	r, err = client.Get(server.URL + "/basic/foo")
	require.NoError(t, err, "simple response cached")
	body, _ = ioutil.ReadAll(r.Body)
	require.Equal(t, "bar", string(body), "full body cached")

	r, err = client.Get(server.URL + "/tpl/debug?param=other-value")
	require.NoError(t, err, "response cached, despite different query params")
	body, _ = ioutil.ReadAll(r.Body)
	require.Equal(t, "\nbar = baz\nparam = foo\n", string(body),
		"body cached, not affected by query parameters")

	_, err = client.Get(server.URL + "/redir")
	require.Equal(t, redirectError, err, "redirects also cached")

	r, err = client.Get(server.URL + "/code/404")
	require.NoError(t, err, "server stopped, response from cache")
	require.Equal(t, r.StatusCode, 404, "status code also cached")

	_, err = client.Get(server.URL + "/code/204")
	require.Error(t, err, "no successful cached response, server stopped")

	_, err = os.Stat(fmt.Sprintf("%s/barista/http/%s_code-204", dir, base))
	require.True(t, os.IsNotExist(err), "transport error not cached")

}
