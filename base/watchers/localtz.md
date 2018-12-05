---
title: watchers/LocalTZ
---

The `localtz` package watches for changes to the machine's time zone, and provides methods to get
the current timezone as well as wait for the timezone to change.

- `Get() *time.Location`: Returns the machine's current time zone. Falls back to `time.Local` if
  timezone tracking is unavailable.

- `Next() <-chan struct{}`: Returns a notifier channel that will be closed the next time the local
  timezone changes. Useful for any displayed local times on the bar.

Additionally, a testonly method `SetForTest(*time.Location)` is also provided to simulate time zone
changes in tests.
