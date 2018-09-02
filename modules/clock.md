---
title: Clock
---

Display local time: `clock.Local()`.  
Display time in a specific timezone: `clock.Zone(time.UTC)` / `clock.ZoneByName("Asia/Tokyo")`.

## Configuration

* `Output(time.Duration, func(time.Time) bar.Output)`: Sets the output format. The first argument
  controls the output granularity. For example, a granularity of 5 seconds would result in the
  output being updated at every 5th second (:00, :05, :10, &hellip;).

* `OutputFormat(string)`: Sets the output format using a time format. The granularity is
  automatically detected based on the input.

* `Timezone(*time.Location)`: Sets the timezone at runtime. This replaces the timezone used when
  constructing the module.

## Examples

<div class="module-example-out">12:25</div>
Formatted time:

```go
clock.Local().OutputFormat("15:04")
```

<div class="module-example-out">4h5m</div>
Using a custom format function:

```go
c, err := clock.ZoneByName("Asia/Tokyo")
// err if loading the timezone by name fails.

c.Output(time.Minute, func(now time.Time) bar.Output {
	return outputs.Textf("%dh%dm", now.Hour(), now.Minute())
})
```

## Data: `time.Time`

[Documentation for time.Time](https://golang.org/pkg/time/#Time)
