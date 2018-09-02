---
title: Counter
---

Create a counter: `counter.New("%d")`.

Counter is a simple counter that can be incremented by scrolling down or right-clicking, and
decremented by scrolling up or left-clicking. It is mostly a demonstration module interactivity.

## Configuration

* `Output(string)`: Sets the output format. The count is passed as an int argument, so `%d` in the
  format will be replaced with the count.
