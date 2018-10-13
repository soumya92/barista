---
title: Static
---

Display static output on the bar: `static.New(outputs.Text("foo"))`.

Static provides a simple module that shows static content on the bar, with
methods to set the content. In a pinch, this can be used to create buttons, or
show additional information by setting the output from within a format function.

## Controlling Static Output

* `Set(bar.Output)`: Sets the new output to be displayed by this module.
* `Clear()`: Clears the output from this module, hiding it from the bar.

## Example

<div class="module-example-out">WWW</div>
Add a button to launch Firefox:

```go
wwwLauncher := static.New(outputs.Text("WWW").OnClick(click.RunLeft("firefox")))
```
