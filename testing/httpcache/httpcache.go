// Copyright 2017 Google Inc.
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

/*
Package httpcache provides a RoundTripper that stores all responses on disk, and
returns cached responses for any requests that have been made before. This
package is provided for a very specific purpose, and is unlikely to be widely
useful. Be aware of the caveats before using this package:
- Requests are keyed purely based on the URL. All other data from the request
  are ignored, which includes headers, POST data, and even query parameters.
- Responses are cached *forever* (until manually deleted).

This cache is useful for quickly prototyping bar customisations. By replacing
the Transport of http.DefaultClient (or replacing http.DefaultTransport), the
resulting binary will no longer make real HTTP requests when started, so it can
be rebuilt and restarted hundreds of times without counting towards any quota.

The cache is located at ~/.cache/barista/http (using XDG_CACHE_HOME for ~/.cache
if set). Individual responses can be deleted if a fresher copy is needed.
*/
package httpcache // import "barista.run/testing/httpcache"

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var sanitizeRe = regexp.MustCompile(`[^[:alnum:]-\.]`)

// keyForURL returns a file-safe key for a given input URL, used as the filename
// for storing cached responses. Because users may want to identify and remove
// individual entries, the key keeps the host and path human-readable.
func keyForURL(u *url.URL) string {
	// Host and URL to make deleting specific cached responses easier.
	return fmt.Sprintf("%s_%s",
		sanitizeRe.ReplaceAllString(u.Host, "-"),
		sanitizeRe.ReplaceAllString(strings.TrimPrefix(u.Path, "/"), "-"))
}

// getCacheDir gets an XDG compliant directory for storing cached responses.
func getCacheDir() string {
	cacheRoot := os.ExpandEnv("$HOME/.cache")
	if xdgCache, ok := os.LookupEnv("XDG_CACHE_HOME"); ok {
		cacheRoot = xdgCache
	}
	return filepath.Join(cacheRoot, "barista", "http")
}

type roundTripper struct {
	http.RoundTripper
	cacheDir string
}

func (c roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	path := filepath.Join(c.cacheDir, keyForURL(req.URL))
	if file, err := os.Open(path); err == nil {
		return http.ReadResponse(bufio.NewReader(file), req)
	}
	resp, err := c.RoundTripper.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	body, err := httputil.DumpResponse(resp, true)
	if err == nil {
		err = ioutil.WriteFile(path, body, 0600)
	}
	return resp, err
}

// Wrap creates a new caching http.RoundTripper that uses the given RoundTripper
// to fetch responses that don't yet exist in the cache.
func Wrap(transport http.RoundTripper) http.RoundTripper {
	cacheDir := getCacheDir()
	os.MkdirAll(cacheDir, 0700)
	return roundTripper{transport, cacheDir}
}
