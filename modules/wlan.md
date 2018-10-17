---
title: Wireless
---

Show wireless information from any interface starting with "wl": `wlan.Any()`.
Show wireless information for a specific interface: `wlan.Named("wlp60s0")`.

In order to fill in additional wireless-only information, this module currently requires the
`/sbin/iwgetid` command, which is usually in a package `wireless-tools`.

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.

## Example

<div class="module-example-out">eduroam (10.4.0.8)</div>
<div class="module-example-out">W: ...</div>
Show the wireless network name and IP:

```go
wlan.Any().Output(func (i wlan.Info) bar.Output {
	switch {
		case !i.Enabled():
			return nil
		case i.Connecting():
			return outputs.Text("W: ...")
		case !i.Connected():
			return outputs.Text("W: down")
		case len(i.IPs) < 1:
			return outputs.Textf("%s (...)", i.SSID)
		default:
			return outputs.Textf("%s (%s)", i.SSID, i.IPs[0])
	}
})
```

## Data: `type Info struct`

### Fields

* `Name string`: Name of the network interface, useful when using `Any()`.
* `State netlink.OperState`: State of the interface. See [netlink#OperState](/base/watchers/netlink#operational-states).
* `IPs []net.IP`: A sorted list of IPs, from global unicast, through broadcast and link-local, to loopback.
* `SSID string`: The human-readable name of the wireless network currently associated with.
* `AccessPointMAC string`: The hardware address of the access point.
* `Channel int`: Channel used by the access point. Ranges vary by frequency.
* `Frequency unit.Frequency`: Frequency of the wireless signals (2.4GHz or 5GHz).

### Methods

* `Enabled() bool`: true if the wireless card is enabled.
* `Connecting() bool`: true if a connection is in progress.
* `Connected() bool`: true if connected to a wireless network.
