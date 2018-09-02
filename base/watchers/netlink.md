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

Because the ordering of updates is significant, **all subscription channels are unbuffered**. To
avoid deadlocks, make sure you call `Unsubscribe` when you're no longer interested in updates.

```go
sub := netlink.Any()
defer sub.Unsubscribe()

for link := range sub {
	// ...
}
```

There is also a `MultiSubscription`, which is similar to `Subscription` except it returns *all*
links on each update. Create one using `netlink.All()`, and unsubscribe using `Unsubscribe()`.

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
