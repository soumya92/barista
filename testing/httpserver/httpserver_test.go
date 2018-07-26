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
	"github.com/stretchr/testify/assert"
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
		assert.Fail(t, "Got error on http.Get(%s): %s", url, e.Error())
		return r, ""
	}
	defer r.Body.Close()
	body, e := ioutil.ReadAll(r.Body)
	if e != nil {
		assert.Fail(t, "Got error while reading response of %s: %s",
			url, e.Error())
	}
	return r, string(body)
}

func testOne(t *testing.T, url string, code int, expected string) {
	resp, body := get(t, url)
	assert.Equal(t, code, resp.StatusCode, "StatusCode for %s", url)
	assert.Equal(t, expected, body, "Body for %s", url)
}

func TestRedir(t *testing.T) {
	ts := New()
	defer ts.Close()
	_, err := http.Get(ts.URL + "/redir")
	assert.Error(t, err, "loop")
}

func TestBasic(t *testing.T) {
	r, body := get(t, "/basic/empty")
	assert.Equal(t, r.StatusCode, 200, "StatusCode for /basic/empty")
	assert.Empty(t, body, "Body for /basic/empty")

	testOne(t, "/basic/foo", 200, "bar")

	testOne(t, "/basic/other", 404, "Not Found")
	testOne(t, "/", 404, "Not Found")
}

func TestHttpCodes(t *testing.T) {
	testOne(t, "/code/404", 404, "Not Found")
	testOne(t, "/code/500", 500, "Internal Server Error")

	r, _ := get(t, "/code/xyz")
	assert.Equal(t, r.StatusCode, 500, "Invalid http code")
}

func TestModTime(t *testing.T) {
	r, _ := get(t, "/modtime/1136239445")
	modTime := r.Header.Get("Last-Modified")
	parsed, err := http.ParseTime(modTime)
	assert.NoError(t, err)
	refTime, _ := time.Parse(time.RubyDate, time.RubyDate)
	assert.WithinDuration(t, refTime, parsed, time.Minute)

	r, _ = get(t, "/modtime/foobar")
	assert.Equal(t, r.StatusCode, 500, "Invalid unix timestamp")
}

func TestTemplate(t *testing.T) {
	fs = afero.NewMemMapFs()
	assert := assert.New(t)

	afero.WriteFile(fs, "testdata/foo.tpl",
		[]byte(`Param1 = {{.param1}}, Param2 = {{.param2}}`), 0400)

	r, body := get(t, "/tpl/foo?param1=abc&param2=xyz")
	assert.Equal(200, r.StatusCode)
	assert.Equal(`Param1 = abc, Param2 = xyz`, body)

	_, body = get(t, "/tpl/foo")
	assert.Equal(`Param1 = , Param2 = `, body, "missing params")

	afero.WriteFile(fs, "testdata/cannot-read.tpl", []byte(`anything`), 0)
	r, _ = get(t, "/tpl/cannot-read?")
	// https://github.com/spf13/afero/issues/150
	// assert.Equal(500, r.StatusCode)

	r, _ = get(t, "/tpl/no-such-template")
	assert.Equal(404, r.StatusCode)

	afero.WriteFile(fs, "testdata/bad.tpl", []byte(`Param = {{.param`), 0400)
	r, _ = get(t, "/tpl/bad")
	assert.Equal(500, r.StatusCode)
}
