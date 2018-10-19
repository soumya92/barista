---
title: group/Modal
---

Create a group that switches between several different "modes".

Start with a new modal: `builder := modal.New()`.

Add new modes, optionally specifying a segment to use in the switcher, and add modules to it:
```go
builder.Mode("mode-name").Output(bar.TextSegment("MN")).
	Add(moduleShownWhenInactiveOrActive).
	Summary(moduleShownOnlyWhenInactive).
	Detail(moduleShownOnlyWhenActive)
```

Once all the modes are set up, build the modal group and get a controller:
`barModule, ctrl := builder.Build()`.

The returned `bar.Module` will display modules according to the active mode (none on start), and
will display a "switcher" at the end of the group. The switcher will show a segment for each mode,
using a text segment with the mode's name if not specified explicitly. Clicking on an inactive mode
will switch to it, clicking on the active mode will revert to the "summary" mode (no active mode).

In summary mode, only the summary modules from each mode are displayed. When a mode is active,
modules from all other modes are hidden, and all detail modules from the active mode are visible.

Modules added using `Add` are shown both in summary and detail modes.

## Example

<div class="module-example-out"><span>d0</span><span>A B %</span></div>
<div class="module-example-out"><span>c0</span><span>c1</span><span>c2</span><span>A B <span class="invert">%</span></span></div>
<div class="module-example-out"><span>b0</span><span>B2</span><span>A <span class="invert">B</span> %</span></div>
<div class="module-example-out"><span>A0</span><span>a1</span><span>a2</span><span>a3</span><span><span class="invert">A</span> B %</span></div>
<div class="module-example-out"><span>A0</span><span>B1</span><span>B2</span><span>A B %</span></div>

Setting up some modes, and switching between them by using the mode switcher:

```go
// For simplicity, assuming mod("a") returns a static module that outputs 'a'.

builder := modal.New()
builder.Mode("A").Add(mod("A0")).Detail(mod("a1"), mod("a2"), mod("a3"))
builder.Mode("B").Detail(mod("b0")).Summary(mod("B1")).Add(mod("B2"))
builder.Mode("C").Detail(mod("c0"), mod("c1"), mod("c2")).Output(bar.TextSegment("%"))
builder.Mode("D").Detail(mod("d0")).Output(nil)

grp, ctrl := builder.Build()
barista.Run(grp)
```
