---
title: testing/HTTPCache
---

The `httpcache` testing package provides a function to wrap an existing `http.RoundTripper` with a
transport that caches responses to disk.

* Requests are keyed purely based on the URL. All other data from the request are ignored, which
  includes headers, POST data, and even query parameters.

* Responses are cached *forever* (until manually deleted).

This cache is useful for quickly prototyping bar customisations. By replacing the `Transport` of
`http.DefaultClient` (or replacing `http.DefaultTransport`), the resulting binary will no longer
make real HTTP requests on each restart, so it can be rebuilt and restarted hundreds of times
without consuming quota.

The cache is located at `~/.cache/barista/http` (using `XDG_CACHE_HOME` for `~/.cache` if set).
Individual responses can be deleted if a fresher copy is needed.

`Wrap(http.RoundTripper) http.RoundTripper`: Returns a new http round tripper that caches responses
to disk, and uses the passed-in round tripper to fetch initial responses.

To use it for a bar binary, it's sufficient to add this to the main file:
```go
import "net/http"
import "barista.run/testing/httpcache"

func init() {
	http.DefaultTransport = httpcache.Wrap(http.DefaultTransport)
}
```
