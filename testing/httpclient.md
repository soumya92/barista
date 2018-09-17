---
title: testing/HTTPClient
---

The `httpclient` testing package provides a function to wrap an existing `*http.Client` with a
transport that redirects all requests to a different host. All headers, paths, query parameters,
cookies, etc. remain the same, only the host+port is rewritten.

`Wrap(*http.Client, string)`: Uses the host from the given base URL string to rewrite all requests
made by the client. This modification occurs in-place. The base URL can be a full URL, only the host
is used. This supports simple usage with an `httptest.Server`:

```go
testServer := httptest.NewServer(/* ... */)
defer testServer.Close()

client := /* get a client */
httpclient.Wrap(client, testServer.URL)

// All further requests will hit testServer instead of the real server.
```
