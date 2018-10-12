---
title: CPU Load Average
---

Display CPU load information: `cpuload.New()`.

The [`sysinfo`](/sysinfo) module can show load averages and additional information, and also shares
updates between all sysinfo module instances, so it should be used over cpuload when possible. 

## Configuration

* `Output(func(LoadAvg) bar.Output)`: Sets the output format.

* `RefreshInterval(time.Duration)`: Sets the interval to wait before fetching updated cpu load
  information. Defaults to 3 seconds.

## Example

<div class="module-example-out">0.30</div>
Show the load average from the last minute:

```go
cpuload.New().Output(func(l LoadAvg) bar.Output) {
	return outputs.Textf("%0.2f", i.Min1())
})
```

## Data: `type LoadAvg [3]float64`

### Methods

* `Min1() float64`: load average for the past 1 minute.
* `Min5() float64`: load average for the past 5 minutes.
* `Min15() float64`: load average for the past 15 minutes.
