---
title: group/Cycling
---

Create a group that cycles through modules: `grp, ctrl := group.Cycling(duration, ...)`.

The returned group is a `bar.Module` and can be added to the bar directly. The second return value
is a `Controller` that can set the refresh interval programmatically. If you only set the interval
during construction, you can ignore it: `grp, _ := group.Cycling(duration, ...)`.

## Example

<div class="module-example-out"><span>c</span></div>
<div class="module-example-out"><span>b</span></div>
<div class="module-example-out"><span>a</span></div>

A simple example of a group that shows each module for 2 seconds.

```go
// For simplicity, assuming a, b, c are simple text modules that show 'a', 'b', and 'c'.
grp, _ := group.Cycling(2 * time.Second, a, b, c)
barista.Run(grp)
```

## Controller

- `SetInterval(time.Duration)`: changes the cycling interval.
