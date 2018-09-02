---
title: Memory Information
---

Display memory information: `meminfo.New()`.

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.

Because the meminfo module performs a single read of `/proc/meminfo` to update all instances, the
refresh interval can only be set for the package as a whole.

* `meminfo.RefreshInterval(time.Duration)`: Sets the interval to wait before fetching updated
  memory information. Defaults to 3 seconds.

## Examples

<div class="module-example-out">0.4/1.8G</div>
Show the used and total swap space:

```go
meminfo.New().Output(func(i meminfo.Info) bar.Output) {
	return outputs.Textf("%.1f/%.1fG",
		i["SwapFree"].Gigabytes(), i["SwapTotal"].Gigabytes())
})
```

<div class="module-example-out">mem:40%/swap:0%</div>
Show the percentage of main memory and swap used:

```go
meminfo.New().Output(func(i meminfo.Info) bar.Output) {
	return outputs.Textf("mem:%.0f%%/swap:%.0f%%",
		(1 - i.FreeFrac("Mem")) * 100.0,
		(1 - i.FreeFrac("Swap")) * 100.0,
	)
})
```

## Data: `type Info map[string]unit.Datasize`

### Keys

Any key in `/proc/meminfo` is also available in Info, parsed into a `unit.Datasize`. The most useful
keys are:
- `MemFree`/`MemTotal`/`MemAvailable`: Main memory
- `SwapFree`/`SwapTotal`: Swap space

### Methods

* `FreeFrac(string) float64`: Fraction free for a given key (e.g. `Mem`, `Swap`, `High`, `Low`), dividing free by total.
* `Available() unit.Datasize`: Available system memory, including cached memory that can be freed up if needed.
* `AvailFrac() float64`: Available memory as a fraction of total.

[Documentation for unit.Datarate](https://godoc.org/github.com/martinlindhe/unit#Datarate)
