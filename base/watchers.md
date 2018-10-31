---
title: base/Watchers
pkg: noimport
---

`base/watchers/` provides push-based watchers for common system events:

- [`dbus`](/base/watchers/dbus): DBus service owners, properties, and signals.

- [`file`](/base/watchers/file): Changes to files, correctly handling entire hierarchies being
  removed or created.

- [`netlink`](/base/watchers/netlink): Changes to network devices and addresses.
