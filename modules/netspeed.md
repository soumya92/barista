---
title: Network Speeds
---

Display network transfer rates for an interface: `netspeed.New("wlan0")`.

## Configuration

* `Output(func(Speeds) bar.Output)`: Sets the output format.

* `RefreshInterval(time.Duration)`: Sets the interval to use when calculating the transfer rates.
  The higher the interval, the smoother the rates, since the rates are based on sampling total data
  transferred at the interval given here.

## Example

<div class="module-example-out">W: 1.4 MiB/s↓ 1.3 KiB/s↑</div>
Show the transfer rates for a wireless interface:

```go
netspeed.New("wlan0").Output(func(s netspeed.Speeds) bar.Output) {
	return outputs.Textf("W: %s↓ %s↑",
		outputs.IByterate(s.Rx), outputs.IByterate(s.Tx))
})
```

## Data: `type Speeds struct`

### Fields

* `Rx unit.Datarate`: Rate of data received by the interface (download).
* `Tx unit.Datarate`: Rate of data transmitted by the interface (upload).

### Methods

* `Total() unit.Datarate`: Total activity of the interface (upload + download).

[Documentation for unit.Datarate](https://godoc.org/github.com/martinlindhe/unit#Datarate)
