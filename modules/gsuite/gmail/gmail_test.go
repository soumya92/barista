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

package gmail

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	"barista.run/testing/httpclient"
	"github.com/stretchr/testify/require"
)

type label struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Total  int    `json:"threadsTotal"`
	Unread int    `json:"threadsUnread"`
}

var (
	labels   map[string]label
	labelsMu sync.Mutex
)

func setLabels(testLabels ...label) {
	labelsMu.Lock()
	defer labelsMu.Unlock()
	labels = map[string]label{}
	for _, lbl := range testLabels {
		labels[lbl.ID] = lbl
	}
}

var fakeClientConfig = []byte(`{
	"installed": {
		"client_id": "143832941570-ek4civ0n1csaahcspkpag91dmfmudd7k.apps.googleusercontent.com",
		"project_id": "i3-barista",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://www.googleapis.com/oauth2/v3/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_secret": "yFSEf5c-vgzzDnfb4vLHqAlr",
		"redirect_uris": ["urn:ietf:wg:oauth:2.0:oob", "http://localhost"]
	}
}`)

func TestEmpty(t *testing.T) {
	testBar.New(t)
	setLabels(label{"INBOX", "INBOX", 15, 0})
	gm := New(fakeClientConfig)
	testBar.Run(gm)
	testBar.NextOutput().AssertEmpty("With no unread messages in inbox")
}

func TestSimple(t *testing.T) {
	testBar.New(t)
	setLabels(
		label{"INBOX", "INBOX", 10, 2},
		label{"label-000", "My label", 2, 0},
		label{"label-001", "Other Label", 0, 0},
	)

	gm := New(fakeClientConfig)
	gm2 := New(fakeClientConfig, "INBOX", "My label")
	testBar.Run(gm, gm2)

	testBar.LatestOutput().AssertText([]string{"Gmail: 2", "Gmail: 2"})

	setLabels(
		label{"INBOX", "INBOX", 15, 7},
		label{"label-000", "My label", 3, 1},
	)
	testBar.Tick()
	testBar.LatestOutput().AssertText([]string{"Gmail: 7", "Gmail: 8"})

	gm.Output(func(i Info) bar.Output {
		return outputs.Textf("%d/%d, %d/%d",
			i.Unread["INBOX"], i.Threads["INBOX"],
			i.TotalUnread(), i.TotalThreads())
	})
	testBar.LatestOutput(0).AssertText([]string{"7/15, 7/15", "Gmail: 8"},
		"Labels not included in module construction are ignored")
}

func TestErrors(t *testing.T) {
	require.Panics(t, func() { New([]byte(`not-a-json-config`)) })

	setLabels(label{"INBOX", "INBOX", 15, 0})
	testBar.New(t)
	gm := New(fakeClientConfig, "no-such-label")
	testBar.Run(gm)
	testBar.NextOutput().AssertError("label not found")

	setLabels( /* empty set will cause an error */ )
	testBar.New(t)
	gm = New(fakeClientConfig)
	testBar.Run(gm)
	testBar.NextOutput().AssertError("error fetching list of labels")
}

func TestMain(m *testing.M) {
	mux := http.NewServeMux()
	mux.HandleFunc("/gmail/v1/users/me/labels", func(w http.ResponseWriter, r *http.Request) {
		labelsMu.Lock()
		defer labelsMu.Unlock()
		if len(labels) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		lbls := struct {
			Labels []label `json:"labels"`
		}{Labels: []label{}}
		for _, l := range labels {
			lbls.Labels = append(lbls.Labels, l)
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(lbls)
	})
	mux.HandleFunc("/gmail/v1/users/me/labels/", func(w http.ResponseWriter, r *http.Request) {
		labelsMu.Lock()
		defer labelsMu.Unlock()
		path := strings.Split(r.URL.Path, "/")
		labelID := path[len(path)-1]
		label, ok := labels[labelID]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(label)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	wrapForTest = func(c *http.Client) {
		httpclient.FreezeOauthToken(c, "authtoken-placeholder")
		httpclient.Wrap(c, server.URL)
	}

	os.Exit(m.Run())
}
