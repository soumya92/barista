---
title: System Information
---

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.

Because the sysinfo module performs a single sysinfo call to update all instances, the refresh
interval can only be set for the package as a whole.

* `sysinfo.RefreshInterval(time.Duration)`: Sets the interval to wait between updating sysinfo.
  Defaults to 3 seconds.

## Example

<div class="module-example-out">up 132h10m78s</div>
Show the system uptime:

```go
sysinfo.New().Output(func(i sysinfo.Info) bar.Output) {
	return outputs.Textf("up %v", i.Uptime)
})
```

## Data: `type Info struct`

### Fields

* `Uptime time.Duration`: System uptime.
* `Loads [3]float64`: Load average for the past 1, 5, and 15 minutes.
* `Procs uint16`: Number of processors (or cores) in the system.
* `TotalRAM unit.Datasize`
* `FreeRAM unit.Datasize`
* `SharedRAM unit.Datasize`
* `BufferRAM unit.Datasize`
* `TotalSwap unit.Datasize`
* `FreeSwap unit.Datasize`
* `TotalHighRAM unit.Datasize`
* `FreeHighRAM unit.Datasize`

[Documentation for unit.Datasize](https://godoc.org/github.com/martinlindhe/unit#Datasize)
