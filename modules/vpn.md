---
title: VPN
---

Show the VPN status for tun0: `vpn.DefaultInterface()`.  
Show the VPN status for a specific interface: `vpn.New("tun1")`.

VPN shows the VPN status as one of three states: Disconnected, Connecting, or Connected. It was
written before netinfo was available, but now a superset of this module's functionality is available
in the netinfo module.

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.

## Example

<div class="module-example-out">VPN</div>
<div class="module-example-out">...</div>
Show a simple "VPN" indicator:

```go
vpn.DefaultInterface().Output(func(s vpn.State) bar.Output {
	if s.Connected() {
		return outputs.Text("VPN")
	}
	if s.Disconnected() {
		return nil
	}
	return outputs.Text("...")
})
```

## Data: `type State int`

### Constants

* `Connected`
* `Connecting`
* `Disconnected`

### Methods

* `Connected() bool`: True if the state is Connected.
* `Disconncted() bool`: True if the state is Disconnected.
