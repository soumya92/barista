---
title: watchers/DBus
import_as: dbusWatcher
---

The `dbus` package provides watchers for dbus properties and signals. All watchers must be cleaned
up when no longer needed, by calling their `Unsubscribe()` method.

## NameOwnerWatcher

A `NameOwherWatcher` watches for changes to the owner for a named service. It supports both single
name and namespaced wildcard names.

- `WatchNameOwner(name string)` watches only the specified service.

- `WatchNameOwners(pattern string)` watches all services within the namespace pattern.

Any updates to relevant service owners will trigger a notification on the `.C` channel, and calling
`GetOwner()` will return the service owner, while `GetOwners()` will return a map of service names
to owners (useful in case of multiple services within the namespace active at the same time).

## PropertiesWatcher

A `PropertiesWatcher` watches for changes to the properties of a DBus object. It signals each change
to the `.Updates` channel, in a map that contains both the previous and the current value for each
updated property.

It also provides `AddSignalHandler(string, func(*Signal, Fetcher) map[string]interface{})` to run
custom signal handlers in case there are properties on the object that trigger signals other than
`PropertiesChanged`. The handler function is provided both the received
[`Signal`](https://godoc.org/github.com/godbus/dbus#Signal), as well as a `Fetcher` to query
additional properties from the object (even properties that are not marked for updates during
construction).

Created using `WatchProperties(...)`:
- `busType BusType`: The bus to use: `Session` or `System`.
- `service string`: The service that exports the object.
- `object dbus.ObjectPath`: The path to the exported object.
- `iface string`: A namespace under which the properties are provided. You can still use other
  properties by providing fully-qualified names, but any short names (no `.`s) will default to this
  interface.
- `properties []string`: A list of all properties of interest. Changes to other properties will be
  ignored.
