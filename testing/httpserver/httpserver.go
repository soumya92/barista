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

// Package httpserver provides a test http server that can serve some
// canned responses, e.g. modification time header, infinite redirect loop,
// various http status codes, and templated responses using query params.
package httpserver // import "barista.run/testing/httpserver"

import (
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
)

var fs = afero.NewOsFs()

// parsePath parses the request path (of the form "/{command}/{arg}")
// into its command and arg components (and returns 404 for other paths).
func parsePath(path string) (command, arg string) {
	parts := strings.SplitN(path, "/", 3)
	command = parts[1]
	if len(parts) > 2 {
		arg = parts[2]
	}
	return command, arg
}

// handleError returns false if error is nil, otherwise writes a 500
// with the error string and returns true.
func handleError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	w.WriteHeader(500)
	w.Write([]byte(err.Error()))
	return true
}

// handleHTTPCode handles the '/code/' path. It parses the arg as an
// http status code, writes a header with that code, and writes the
// corresponding message in the body (e.g. '/code/404' => 'Not Found').
func handleHTTPCode(w http.ResponseWriter, arg string) {
	code, err := strconv.ParseInt(arg, 10, 32)
	if handleError(w, err) {
		return
	}
	w.WriteHeader(int(code))
	w.Write([]byte(http.StatusText(int(code))))
}

// handleRedirect handles the '/redir/' path. It redirects to itself,
// using HTTP 307.
func handleRedirect(w http.ResponseWriter, urlPath string) {
	w.Header().Set("Location", urlPath)
	w.WriteHeader(307)
}

// handleModTime handles the '/modtime/' path. It sets the last modified
// header to the unix timestamp parsed from arg.
func handleModTime(w http.ResponseWriter, arg string) {
	ts, err := strconv.ParseInt(arg, 10, 64)
	if handleError(w, err) {
		return
	}
	w.Header().Set("Last-Modified",
		time.Unix(ts, 0).In(time.UTC).Format(http.TimeFormat))
	w.Write([]byte("This page was modified at unix " + arg))
}

// handleBasic handles two special files, '/basic/foo' and '/basic/empty'.
// '/basic/foo' returns 'bar', while '/basic/empty' is empty.
func handleBasic(w http.ResponseWriter, arg string) {
	switch arg {
	case "foo":
		w.WriteHeader(200)
		w.Write([]byte("bar"))
	case "empty":
		w.WriteHeader(200)
	default:
		handleHTTPCode(w, "404")
	}
}

// handleStatic handles the '/static/' path. It treats arg as the name of
// a file to load, relative to "testdata".
func handleStatic(w http.ResponseWriter, arg string) {
	file, err := fs.Open(filepath.Join("testdata", arg))
	if os.IsNotExist(err) {
		handleHTTPCode(w, "404")
		return
	}
	if handleError(w, err) {
		return
	}
	w.WriteHeader(200)
	io.Copy(w, file)
	file.Close()
}

// handleTemplate handles the '/tpl/' path. It treats arg as the name of
// the template file to load, relative to "testdata", and passes all query
// parameters as arguments to the template.
func handleTemplate(w http.ResponseWriter, arg string, queryParams url.Values) {
	tplFileName := filepath.Join("testdata", arg+".tpl")
	tplFile, err := afero.ReadFile(fs, tplFileName)
	if os.IsNotExist(err) {
		handleHTTPCode(w, "404")
		return
	}
	if handleError(w, err) {
		return
	}
	t, err := template.New("response").Parse(string(tplFile))
	if handleError(w, err) {
		return
	}
	data := map[string]string{}
	for k, vs := range queryParams {
		data[k] = vs[len(vs)-1]
	}
	handleError(w, t.Execute(w, data))
}

// New creates a new test server with some pre-configured special routes.
func New() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cmd, arg := parsePath(r.URL.Path)
		switch cmd {
		case "code":
			handleHTTPCode(w, arg)
		case "redir":
			handleRedirect(w, r.URL.Path)
		case "modtime":
			handleModTime(w, arg)
		case "basic":
			handleBasic(w, arg)
		case "static":
			handleStatic(w, arg)
		case "tpl":
			handleTemplate(w, arg, r.URL.Query())
		default:
			handleHTTPCode(w, "404")
		}
	}))
}
