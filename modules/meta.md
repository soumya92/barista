---
title: Meta Modules
pkg: 'noimport'
---

`modules/meta` provides modules that operate on other modules.

* [reformat](/modules/meta/reformat): Transforms the output from an existing module before
  sending it to the bar.

* [split](/modules/meta/split): Splits the output from an existing module and returns two
  separate modules.

* [multicast](/modules/meta/multicast): Allows a module to be added to the bar multiple times.
  Especially useful when using groups.

* [slot](/modules/meta/slot): Allows a module to occupy named "slots", with the active slot being
  changeable at runtime, allowing limited repositioning of module output.
