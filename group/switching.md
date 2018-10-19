---
title: group/Switching
---

Create a group that shows one module at a time: `grp, ctrl := switching.Group(...)`.

The returned group is a `bar.Module` and can be added to the bar directly. The second return value
is a `Controller` that provides methods to control the group programatically. If you only use the
built-in buttons to control the group, it can be safely ignored: `grp, _ := switching.Group(...)`.

## Example

<div class="module-example-out"><span>-</span><span>c</span></div>
<div class="module-example-out"><span>-</span><span>b</span><span>+</span></div>
<div class="module-example-out"><span>a</span><span>+</span></div>

A simple example of a collapsing group, with custom buttons.

```go
// For simplicity, assuming a, b, c are simple text modules that show 'a', 'b', and 'c'.
grp, ctrl := switching.Group(a, b, c)
// The button function receives the current index and the count.
ctrl.ButtonFunc(func(index, count int) (start, end bar.Output) {
	if index + 1 < count {
		end = outputs.Text("+")
	}
	if index > 0 {
		start = outputs.Text("-")
	}
	return // start, end
})

barista.Run(grp)
```

## Controller

- `Current() int`: returns the index of the currently active module.
- `Previous()`: switches to the previous module.
- `Next()`: switches to the next module.
- `Show(int)`: sets the currently active module.
- `Count()`: returns the number of modules in this group
- `ButtonFunc(ButtonFunc)`: controls the output for the buttons on either end.
