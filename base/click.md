---
title: base/Click
---

The `click` package provides helpers to make it easier to add click handlers to segments.

* `DiscardEvent(func()) func(bar.Event)`: This takes a `func()` and creates a
  `func(bar.Event)`, allowing simple functions to be used as click handlers
  without needing to write out the wrapping function at each call site:

  ```go
out.OnClick(click.DiscardEvent(g.Toggle))
```
  
  However, this will cause the function to be called on *any* event, including
  mouse wheel events. To restrict it to buttons, use the `Click` function.

* `Click(func(), bool) func(bar.Event)`: Takes a `func()` and creates a click
  handler that invokes it only on button clicks (left, right, and middle). An
  optional second parameter can be set to true to also include back and forward
  buttons:

  ```go
out.OnClick(click.Click(g.Next, true))
```

* `RunLeft(cmd string, args ...string)`: Takes a command and arguments, and
  executes it when left clicked:

  ```go
out.OnClick(click.RunLeft("xdg-open", "/"))
```

* `Scroll(func(bar.Button)) func(bar.Event)`: Calls a function only on scroll
  events, and passes in the `bar.Button` to allow the function to distinguish
  between scroll directions.

  You can also use the `ScrollUp`, `ScrollDown`, `ScrollRight`, and `ScrollLeft`
  functions to call a `func()` for each direction separately.

## Map

`click.Map` is a map of `bar.Button` to `func(bar.Event)`, such that each function
is only called when the particular button triggers an event. It also provides a
fallback handler using `Else` that can catch all remaining events.

```go
out.OnClick(click.Map{
	bar.ButtonLeft: func(bar.Event) { /* ... */ },
	bar.ButtonRight: func(bar.Event) { /* ... */ },
}.Handle)
```

For a more chainable API, Map adds several functions that return the Map itself:

* `Set(bar.Button, func(bar.Event)) Map`: Sets the handler for a button.

* `Else(func(bar.Event)) Map`: Sets the fallback handler, invoked when nothing
  was previously specified for the button that triggered an event.

* `Left(func()) Map`: Sets the left-click handler to a simple function, a shortcut
  for `Set(bar.ButtonLeft, DiscardEvent(/* ... */))`.

  Similar methods exist for all buttons: `Right`, `Middle`, `Back`, `Forward`,
  and scroll directions: `ScrollUp`, `ScrollDown`, `ScrollLeft`, `ScrollRight`.
