---
title: group/Following
---

Create a group that shows the most recently updated module: `grp := following.Group(...)`.

Avoid using modules that refresn at a fixed interval, they may refresh at the same time. Instead,
this is most useful for async modules, e.g. media player or window title.

## Example

<div class="module-example-out"><span>b</span></div>
<div class="module-example-out"><span>c</span></div>
<div class="module-example-out"><span>a</span></div>

A simple example of a following group that shows the most recently updated module.

```go
grp := following.Group(a, b, c)
barista.Run(grp)
```
