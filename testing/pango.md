---
title: testing/Pango
import_as: pangoTest
---

The `pango` testing package provides methods to assert equality of pango markup strings in a markup
aware fashion (ignoring non-printing whitespace, and comparing attributes without regard for order).

## Example

```go
pangoTesting.AssertEqual(t,
	"<span face='monospace' color='#f00'>red</span>",
	`<span color='#f00'
	       face='monospace'>red</span>`)
```

See the [tests](https://github.com/soumya92/barista/blob/master/testing/pango/pango_test.go) for
more examples.
