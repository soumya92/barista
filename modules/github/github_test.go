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

package github

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	"barista.run/testing/httpclient"
	"barista.run/timing"
	"github.com/stretchr/testify/require"
)

var (
	responseFunc   func(http.ResponseWriter, *http.Request)
	responseFuncMu sync.Mutex
)

func respondWith(fn func(http.ResponseWriter, *http.Request)) {
	responseFuncMu.Lock()
	defer responseFuncMu.Unlock()
	responseFunc = fn
}

func respondWithSuccess(contents string, lastModified string, pollInterval string) {
	respondWith(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Last-Modified", lastModified)
		w.Header().Add("X-Poll-Interval", pollInterval)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, contents)
	})
}

func TestSimple(t *testing.T) {
	testBar.New(t)

	respondWith(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, "[]")
	})

	gh := New()
	testBar.Run(gh)

	testBar.NextOutput().AssertEmpty("with no notifications")
	now := timing.Now()
	testBar.Tick()
	testBar.NextOutput().AssertEmpty("with no notifications")

	require.WithinDuration(t, timing.Now(), now.Add(10*time.Second), time.Millisecond,
		"Default delay is 10s when no header is given")

	respondWithSuccess("[]", "", "96")
	testBar.Tick()
	testBar.NextOutput().AssertEmpty("with no notifications")

	now = timing.Now()
	testBar.Tick()
	testBar.NextOutput().AssertEmpty("with no notifications")

	require.WithinDuration(t, timing.Now(), now.Add(96*time.Second), time.Millisecond,
		"Delays from X-Poll-Interval is honoured")

	respondWithSuccess(`[
{"reason": "mention", "unread": true},
{"reason": "mention", "unread": true},
{"reason": "mention", "unread": true},
{"reason": "following", "unread": true}
]`, "Thu, 25 Oct 2012 15:16:27 GMT", "45")

	testBar.Tick()
	testBar.NextOutput().AssertText([]string{"GH: 4"})

	gh.Output(func(n Notifications) bar.Output {
		return outputs.Textf("M:%d,F:%d", n["mention"], n["following"])
	})
	testBar.NextOutput().AssertText([]string{"M:3,F:1"},
		"on output format change")

	respondWith(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Thu, 25 Oct 2012 15:16:27 GMT", r.Header.Get("If-Modified-Since"),
			"Last-Modified header is passed along")
		w.WriteHeader(http.StatusNotModified)
	})
	testBar.Tick()
	testBar.AssertNoOutput("On 304")

	gh.Output(func(n Notifications) bar.Output {
		return outputs.Textf("GH:%d", n.Total())
	})
	testBar.NextOutput().AssertText([]string{"GH:4"},
		"keeps previous value on cached response")

	respondWithSuccess(`[
{"reason": "mention", "unread": true},
{"reason": "mention", "unread": false},
{"reason": "mention", "unread": true},
{"reason": "following", "unread": true}
]`, "", "")
	testBar.Tick()
	testBar.NextOutput().AssertText([]string{"GH:3"}, "Skips read notifications")
}

func TestErrors(t *testing.T) {
	testBar.New(t)

	respondWith(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	gh := New()
	testBar.Run(gh)

	err := testBar.NextOutput().AssertError("On HTTP Error")
	require.Contains(t, err, "HTTP Status 403")

	respondWithSuccess("not-valid-json", "", "")
	testBar.NextOutput().At(0).LeftClick()
	testBar.NextOutput().Expect("on restart")

	testBar.NextOutput().AssertError("On JSON Error")

	respondWith(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "/notifications")
		w.WriteHeader(http.StatusTemporaryRedirect)
	})
	testBar.NextOutput().At(0).LeftClick()
	testBar.NextOutput().Expect("on restart")
	testBar.NextOutput().AssertError("On HTTP Client Error")
}

func TestMain(m *testing.M) {
	mux := http.NewServeMux()
	mux.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		responseFuncMu.Lock()
		defer responseFuncMu.Unlock()
		responseFunc(w, r)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	wrapForTest = func(c *http.Client) {
		httpclient.FreezeOauthToken(c, "authtoken-placeholder")
		httpclient.Wrap(c, server.URL)
	}

	os.Exit(m.Run())
}
