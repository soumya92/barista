---
title: Volume
---

Show volume for an ALSA device: `volume.Mixer(card, mixer string)` / `volume.DefaultMixer()`.  
Show volume for a PulseAudio device (using D-Bus): `volume.Sink(string)` / `volume.DefaultSink()`.

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.
  
  If a segment does not have a click handler, the module will set a default click handler, which:
  - Toggles mute on left click
  - Increases/decreases the volume on scroll up/down

## Example

<div class="module-example-out">V:065</div>
<div class="module-example-out">V:MUT</div>
Show the volume percentage (or "MUT"):

```go
volume.DefaultMixer().Output(func(v volume.Volume) bar.Output {
	if v.Mute {
		return outputs.Text("V:MUT")
	}
	return outputs.Textf("V:%03d", v.Pct())
})
```

## Data: `type Volume struct`

### Fields

* `Min, Max int64`: Minimum/Maximum valid values for volume.
* `Vol int64`: Current volume.
* `Mute bool`: Whether the volume is currently muted. Vol may be non-zero even if this is set.

### Methods

* `Frac() float64`: Volume as a fraction of its range (0.0 = min, 1.0 = max).
* `Pct() int`: Volume as an integral percentage of its range (0 = min, 100 = max).

### Controller Methods

In addition to the data methods listed above, volume also provides controller methods to set the 
volume state:

* `SetMuted(bool)`: Sets the mute state.
* `SetVolume(int64)`: Sets the volume, clamped to Min/Max.
