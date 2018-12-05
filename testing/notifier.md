---
title: testing/Notifier
---

The `notifier` testing package provides methods to assert that a notifier `<- chan struct{}` was
correctly triggered (or not).

## Methods

- `AssertNotified(*testing.T, <-chan struct{}, ...)`: Asserts that an empty struct was received on
  the notifier channel.

- `AssertClosed(*testing.T, <-chan struct{}, ...)`: Asserts that the notifier channel was closed.

- `AssertNoUpdate(*testing.T, <-chan struct{}, ...)`: Asserts that the channel was not updated in
  any way (i.e. a `select` would not trigger). This means the channel received no value, and was not
  closed.
