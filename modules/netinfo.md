---
title: Network Information
---

Display network information for the "best" interface: `netinfo.New()`.  
Display network information for the "best" interface with a prefix: `netinfo.Prefix("wl")`.  
Display network information for a specific interface: `netinfo.Interface("eno1")`.

The "best" network interface is selected by the state. A connected interface is preferred, going
down through various states (Dormant, Down, NotPresent), until Unknown, which is only used if
nothing else is available.

## Configuration

* `Output(func(State) bar.Output)`: Sets the output format.

## Example

<div class="module-example-out">eno1: 10.2.0.1</div>
Show the most relevant IP from any interface:

```go
netinfo.New().Output(func(s netinfo.State) bar.Output) {
	if len(s.IPs) < 1 {
		return outputs.Text("No network").Color(colors.Scheme("bad"))
	}
	return outputs.Textf("%s: %v", s.Name, s.IPs[0])
})
```

## Data: `type State struct`

### Fields

* `Name string`: Name of the interface, e.g. "eno1".
* `State netlink.OperState`: State of the interface. See [netlink#OperState](/base/watchers/netlink#operational-states).
* `HardwareAddr net.HardwareAddr`: Hardware address of the interface (a.k.a. MAC address).
* `IPs []net.IP`: A sorted list of IPs, from global unicast, through link-local and multicast, to
  loopback. See the [netlink IP address docs](/base/watchers/netlink#ip-addresses) for exact priorities of IP addresses.

### Methods

* `Connecting() bool`: Returns true if a connection is in progress.
* `Connected() bool`: Returns true if connected to a network.
* `Enabled() bool`: Returns true if a network interface is enabled.

Documentation for [net.IP](https://golang.org/pkg/net/#IP) and
[net.HardwareAddr](https://golang.org/pkg/net/#HardwareAddr)
