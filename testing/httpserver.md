---
title: testing/HTTPServer
---

The `httpserver` testing package provides an HTTP server that exhibits certain special behaviours
when given certain URLs.

* `/code/$code`: A plain-text response with the numeric code in the header, and a description (e.g.
  "Not Found") in the body.

* `/redir`: An infinite loop of redirection.

* `/modtime/$unixtime`: A text response with a Last-Modified header set to the given unix time.

* `/basic/foo`: Plain text "bar".

* `/basic/empty`: A 200 response with no body.

* `/static/$file`: Contents of "testdata/$file".

* `/tpl/$file`: The template from "testdata/$file.tpl", evaluated with the URL parameters.

