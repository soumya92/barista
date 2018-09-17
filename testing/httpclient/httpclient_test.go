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
	"net/http"
	"net/http/httptest"
	"testing"

	"barista.run/testing/httpserver"
	"github.com/stretchr/testify/require"
)

func TestWrapper(t *testing.T) {
	okServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
}
