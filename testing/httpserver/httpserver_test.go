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

package httpserver

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

var ts *httptest.Server

func TestMain(m *testing.M) {
	ts = New()
	defer ts.Close()
	os.Exit(m.Run())
}

func get(t *testing.T, url string) (*http.Response, string) {
	r, e := http.Get(ts.URL + url)
	if e != nil {
		require.Fail(t, "Got error on http.Get(%s): %s", url, e.Error())
		return r, ""
	}
	defer r.Body.Close()
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		require.Fail(t, "Got error while reading response of %s: %s",
			url, e.Error())
	}
	return r, string(body)
}

func testOne(t *testing.T, url string, code int, expected ...string) {
	resp, body := get(t, url)
	require.Equal(t, code, resp.StatusCode, "StatusCode for %s", url)
	if len(expected) == 1 {
		require.Equal(t, expected[0], body, "Body for %s", url)
	}
}

func TestRedir(t *testing.T) {
	ts := New()
	defer ts.Close()
	_, err := http.Get(ts.URL + "/redir")
	require.Error(t, err, "loop")
}

func TestBasic(t *testing.T) {
	r, body := get(t, "/basic/empty")
	require.Equal(t, r.StatusCode, 200, "StatusCode for /basic/empty")
	require.Empty(t, body, "Body for /basic/empty")

	testOne(t, "/basic/foo", 200, "bar")

	testOne(t, "/basic/other", 404, "Not Found")
	testOne(t, "/", 404, "Not Found")
}

func TestHttpCodes(t *testing.T) {
	testOne(t, "/code/404", 404, "Not Found")
	testOne(t, "/code/500", 500, "Internal Server Error")
	testOne(t, "/code/xyz", 500)
}

func TestModTime(t *testing.T) {
	r, _ := get(t, "/modtime/1136239445")
	modTime := r.Header.Get("Last-Modified")
	parsed, err := http.ParseTime(modTime)
	require.NoError(t, err)
	refTime, _ := time.Parse(time.RubyDate, time.RubyDate)
	require.WithinDuration(t, refTime, parsed, time.Minute)

	testOne(t, "/modtime/foobar", 500)
}

func TestStatic(t *testing.T) {
	fs = afero.NewMemMapFs()
	require := require.New(t)

	afero.WriteFile(fs, "testdata/foo", []byte(`bar`), 0400)
	testOne(t, "/static/foo", 200, "bar")

	afero.WriteFile(fs, "testdata/empty", []byte{}, 0400)
	r, body := get(t, "/static/empty")
	require.Equal(200, r.StatusCode)
	require.Empty(body)

	testOne(t, "/static/not-found", 404)

	afero.WriteFile(fs, "testdata/no-read", []byte{}, 0)
	// https://github.com/spf13/afero/issues/150
	// testOne(t, "/static/no-read", 500)
}

func TestTemplate(t *testing.T) {
	fs = afero.NewMemMapFs()

	afero.WriteFile(fs, "testdata/foo.tpl",
		[]byte(`Param1 = {{.param1}}, Param2 = {{.param2}}`), 0400)

	testOne(t, "/tpl/foo?param1=abc&param2=xyz", 200, `Param1 = abc, Param2 = xyz`)
	testOne(t, "/tpl/foo", 200, `Param1 = , Param2 = `)

	afero.WriteFile(fs, "testdata/cannot-read.tpl", []byte(`anything`), 0)
	// https://github.com/spf13/afero/issues/150
	// testOne(t, "/tpl/cannot-read?", 500)

	testOne(t, "/tpl/no-such-template", 404)

	afero.WriteFile(fs, "testdata/bad.tpl", []byte(`Param = {{.param`), 0400)
	testOne(t, "/tpl/bad", 500)
}
