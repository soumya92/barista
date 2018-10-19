---
title: group/Collapsing
---

Create a group that can be expanded/collapsed with a button: `grp, ctrl := collapsing.Group(...)`.

The returned group is a `bar.Module` and can be added to the bar directly. The second return value
is a `Controller` that provides methods to control the group programatically. If you only use the
built-in buttons to control the group, it can be safely ignored: `grp, _ := collapsing.Group(...)`.

## Example

<div class="module-example-out"><span>[expand]</span></div>
<div class="module-example-out"><span>a</span><span>b</span><span>c</span><span>[collapse]</span></div>

A simple example of a collapsing group, with custom buttons.

```go
// For simplicity, assuming a, b, c are simple text modules that show 'a', 'b', and 'c'.
grp, ctrl := collapsing.Group(a, b, c)
// By default the group starts collapsed, but we can use the controller to expand it.
ctrl.Expand()
// We can also change the default buttons
ctrl.ButtonFunc(func(expanded bool) (start, end bar.Output) {
	if expanded {
		// Note that if we don't provide a click handler, the default handler will be used.
		return nil, outputs.Text("[collapse]")
	}
	return nil, outputs.Text("[expand]")
})

barista.Run(grp)
```

## Controller

- `Expanded() bool`: returns true if the group is expanded.
- `Collapse()`: collapses the group, hides all modules.
- `Expand()`: expands the group, shows all modules.
- `Toggle()`: toggles the visibility of all modules.
- `ButtonFunc(ButtonFunc)`: controls the output for the button(s).
