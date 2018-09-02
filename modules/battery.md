---
title: Battery
---

Display information about a specific battery: `battery.Named("BAT0")`.  
Display aggregated information for all available batteries: `battery.All()`.

Aggregated information effectively creates a "virtual" battery where the stats are sensibly merged
or computed across all available batteries. For example, the Energy Now is just a sum of all
batteries, but the Voltage is a weighted average (by max energy), while Technology is a
comma-separated merger of individual battery technologies.

## Configuration

* `Output(func(battery.Info) bar.Output)`: Sets the output format.

* `RefreshInterval(time.Duration)`: Sets the interval to wait before fetching updated battery
  information. Defaults to 3 seconds.

## Examples

<div class="module-example-out">4h33m</div>
Show time remaining on battery (or to full charge if plugged in):

```go
battery.All().Output(func(i battery.Info) bar.Output) {
	return outputs.Text(i.RemainingTime().String())
})
```

<div class="module-example-out">46Wh+13W</div>
<div class="module-example-out">97Wh-21W</div>
Show the energy stored and usage rate for a specific battery:

```go
battery.Named("BAT0").Output(func(i battery.Info) bar.Output {
	sep := "+"
	if i.Discharging() {
		sep = "-"
	}
	return outputs.Textf("%.0fWh%s%.0fW", i.EnergyNow, sep, i.Power)
})
```

## Data: `type Info struct`

### Fields

* `Capacity int`: Capacity in percents, from 0 to 100.
* `EnergyFull float64`: Energy when the battery is full, in Wh.
* `EnergyMax float64`: Max Energy the battery can store, in Wh.
* `EnergyNow float64`: Energy currently stored in the battery, in Wh. 
* `Power float64`: Power currently being drawn from the battery, in W.
* `Voltage float64`: Current voltage of the batter, in V.
* `Status Status`: Status of the battery, e.g. "Charging", "Full", "Disconnected".
* `Technology string`: Technology of the battery, e.g. "Li-Ion", "Li-Poly", "Ni-MH".

### Methods

* `Remaining() float64`: fraction of battery capacity remaining.
* `RemainingPct() int`: percentage of battery capacity remaining.
* `RemainingTime() time.Duration`: approximate remaining time, based on the current power draw and remaining capacity.
* `Discharging() bool`: true if the battery is being discharged.
* `PluggedIn() bool`: true if the battery is plugged in.
* `SignedPower() float64`: returns the current power in W, positive if being charged, negative if being discharged.
