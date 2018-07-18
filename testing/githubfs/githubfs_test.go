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

package githubfs

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	assert.Contains(t, New().Name(), "GitHubFS")
}

func TestFs(t *testing.T) {
	oldTime := time.Now().Add(-127 * time.Hour)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/loop":
			w.Header().Set("Location", "/loop")
			w.WriteHeader(307)
		case "/500":
			w.WriteHeader(500)
			w.Write([]byte("Something went wrong"))
		case "/oldfile":
			w.Header().Set("Last-Modified",
				oldTime.In(time.UTC).Format(http.TimeFormat))
			w.Write([]byte("foo"))
		case "/empty":
			w.Write([]byte{})
		case "/foo":
			w.Write([]byte("bar"))
		}
	}))
	defer ts.Close()
	root = ts.URL

	fs := New()

	_, err := fs.Open("500")
	assert.Error(t, err)
	_, err = fs.OpenFile("/loop", 0, 0444)
	assert.Error(t, err)
	_, err = fs.Stat("500")
	assert.Error(t, err)

	info, err := fs.Stat("oldfile")
	assert.NoError(t, err)
	assert.Equal(t, oldTime.Truncate(time.Second), info.ModTime())

	f, err := fs.Open("empty")
	assert.NoError(t, err)
	contents, err := ioutil.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, contents)

	f, err = fs.OpenFile("foo", os.O_RDONLY, 0600)
	assert.NoError(t, err)
	contents, err = ioutil.ReadAll(f)
	assert.NoError(t, err)
	assert.Equal(t, "bar", string(contents))
}
