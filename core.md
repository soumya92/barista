---
title: Core
---

Core provides the building blocks for barista. Extracting the core functionality into a smaller
package allows more flexible code. For examples, see the [`reformat` module](/modules/reformat)
and the [`group` package](/group).

## core.Module

`NewModule(bar.Module)` creates a `core.Module` from a `bar.Module`. A core Module decorates the
original module by adding:

* `Stream(core.Sink)`

  Instead of a bar.Sink, Stream accepts a core.Sink, making the body of the sink simpler by
  eliminating some nil checks. A core Module converts `nil` output from the original module to an
  empty slice, while all other outputs are converted to their segment slices.

* Restart click handlers
  
  When the original module's `Stream` method returns, a core Module replays the last output but sets
  the click handlers to a function that restarts the original module (and hides any error segments).

* `Replay()`
  
  Apart from the restart case, a core.Module also exposes a `Replay()` method that just replays the
  last output from the original module.

## core.ModuleSet

`NewModuleSet([]bar.Module)` creates a `core.ModuleSet` from a slice of `bar.Module`s.

* `Stream()`: Starts streaming the original modules, and returns an **unbuffered** `<-chan int`.
  The returned channel will be given the index of the module anytime there is new output from the
  module, at which point `LastOutput(int)` or `LastOutputs()` can be used to get the most recent
  output from one or all of the modules.

* `LastOutput(int)`: Gets the last output from the module at the given index. This is returned as
  `bar.Segments`, so nil checks are not necessary.

* `LastOutputs()`: Returns a `[]bar.Segments` with the last slice of segments emitted by each module
  in the set.
