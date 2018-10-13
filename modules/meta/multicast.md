---
title: Multicast
---

Most modules can only be added once, since they maintain internal state.
Multicast provides a simple wrapper that allows multiple copies. Combined with
[`group.Simple()`](/group#groupsimple), this can create many different
combinations of modules without needing multiple instances of any module.

Multicasting a module: `multi := multicast.New(existingModule)`.

*Important*: The original module must NOT be added to the bar. Only the returned
module can be added, although it can be added multiple times.

## Example

<div class="module-example-out"><span>&lt;</span><span>00:40</span><span>35 deg</span><span>845 MiB</span><span>&gt;</span></div>
<div class="module-example-out"><span>&lt;</span><span>00:40</span><span>Mon, Jan 21</span><span>35 deg</span><span>&gt;</span></div>
Using a switching group to show different bar layouts:

```go
// simple modules
date := clock.Local().Format("Mon, Jan 2")
mem := sysinfo.New().Output(/* free memory */)

// multicasted modules
time := multicast.New(clock.Local().Format("15:04"))
wthr := multicast.New(weather.New(/* provider */).Output(/* formatter */))

layoutA := group.Simple(time, wthr, mem)
layoutB := group.Simple(time, date, wthr)
layoutC := group.Simple(time, wthr)

grp, _ := switching.Group(layoutA, layoutB, layoutC)
barista.Run(grp)
```
