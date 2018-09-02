---
title: CPU Temperature
---

Display the temperature of the first thermal sensor: `cputemp.DefaultZone()`.  
Display the temperature of a specific sensor: `cputemp.Zone("thermal_zone7")`.

## Configuration

* `Output(func(unit.Temperature) bar.Output)`: Sets the output format.

* `RefreshInterval(time.Duration)`: Sets the interval to wait before fetching updated temperature
  information. Defaults to 3 seconds. If the temperature has not changed, the output function will
  not be called again.

## Example

<div class="module-example-out">25 deg C</div>
Show the temperature from zone 7 in celsius:

```go
cputemp.Zone("thermal_zone7").Output(func(t unit.Temperature) bar.Output) {
	return outputs.Textf("%.0f deg C", t.Celsius())
})
```

## Data: `unit.Temperature`

[Documentation for unit.Temperature](https://godoc.org/github.com/martinlindhe/unit#Temperature)
