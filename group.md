---
title: Groups
---

The `group` package provides a base for grouping modules and selectively displaying them on the bar.
There are several default groups:

* [Collapsing](/group/collapsing): Show/hide all the modules in the group.
* [Switching](/group/switching): Switch between modules using the buttons on the end.
* [Cycling](/group/cycling): Cycle through the modules one at a time at a fixed time interval.
* [Following](/group/following): Show the most recently updated module.
* [Modal](/group/modal): Switch between multiple "modes" of the bar.

## group.Simple

In addition to the more functional default groups, barista also includes `group.Simple(...)` that
simply concatenates several modules into one:

```go
newModule = group.Simple(mod1, mod2, mod3)
```

Allowing multiple modules to be added anywhere a single module is expected, for example, in the
cycling or switching groups.

## Implementing a Custom Group

If none of the built-in groupers are a perfect fit, you can also write your own grouper. The basic
interface is

```go
// Grouper controls how a group displays the output from it's modules.
type Grouper interface {
	Visible(index int) bool
	Buttons() (start, end bar.Output)
}
```

* `Visible(int) bool`: Will be called once for each index. Any modules for which Visible returns
  false will not be shown on the bar.

* `Buttons() (start, end bar.Output)`: Called once for each update. The buttons returned will be
  shown on either end of the group as a whole. `nil` values are valid, and will hide the button.

Additional interfaces, if implemented, provide even more control over the output:

```go
type UpdateListener interface {
	Updated(index int)
}
```

If the grouper implements UpdateListener, the `Update(int)` method will be called with the index of
the module that was most recently updated.

```go
type Signaller interface {
	Signal() <-chan struct{}
}
```

If the grouper implements Signaller, the `Signal()` method will be called during the initial stream,
and any updates the the returned channel will cause the group to recalculate the displayed output
using the last output from each module.
