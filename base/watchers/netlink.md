---
title: watchers/Netlink
---

The `netlink` package provides a shared watcher for network link events. It allows clients to create
filtered subscriptions for one or more links, and streams updates for each link to all interested
clients.

There are several methods to create a subscription:

- `ByName(string)`: Subscribes to updates for a specific link.

- `WithPrefix(string)`: Subscribes to updates for all links with the given prefix. However, only one
  link will be sent over the subscription channel, and it will be the "best" link that has this
  prefix. For how links are prioritised, see [operational states](#operational-states).

- `Any()`: Subscribes to updates for any link. The best link after any update will be sent to the
  channel for this subscription.

Call `Unsubscribe` when you're no longer interested in updates. Because the order of updates is
significant, the netlink watcher internally applies updates as soon as they're received, and uses
a notifier on `C` to signal changes.

At any point, the `Get()` will return the most appropriate link based on the subscription criteria.

```go
sub := netlink.Any()
defer sub.Unsubscribe()

for range sub.C {
	link := sub.Get()
	// ...
}
```

There is also a `MultiSubscription`, returned by `netlink.All()`, that hooks into the global netlink
listener directly. No unsubscribe call is necessary, since all `MultiSubscription`s are a view on
the same backing data that powers other netlink subscriptions.

## `type Link struct`

### Fields

- `Name string`: 
- `State OperState`: 
- `HardwareAddr net.HardwareAddr`: 
- `IPs []net.IP`: 

## Operational States

Links are prioritised based on their operational state. In decreasing priority, the operation states
used by this package are:

- `Up`: Link is connected
- `Dormant`: Link is waiting for a connection to be established
- `Testing`: Link is in testing mode
- `LowerLayerDown`: Composite/virtual link, and a required link is down
- `Down`: Link is down
- `NotPresent`: Link is not present
- `Unknown`: No status available
- `Gone`: Link was previously available, but is no longer present

## IP Addresses

Within a link, the IP addresses are also sorted, so that .IPs[0] is the "best" IP address. IP
addresses are prioritised as below:

- `GlobalUnicast`
- `LinkLocalUnicast`
- `LinkLocalMulticast`
- `InterfaceLocalMulticast`
- `Multicast`
- `Loopback`
