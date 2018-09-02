---
title: Disk IO
---

Display disk I/O rates for a device: `diskio.New("sda1")`.

The disk names used are the ones shown in the third column of `/proc/diskstats`.

## Configuration

* `Output(func(IO) bar.Output)`: Sets the output format.

Because the diskio module performs a single read to update all instances, the refresh interval can
only be set for the package as a whole.

* `diskio.RefreshInterval(time.Duration)`: Sets the interval to wait between updating disk stats.
  Defaults to 3 seconds. On each refresh of disk information, all diskio modules will be updated.

## Example

<div class="module-example-out">sda1: 5.0 KiB/s</div>
Show the total disk activity for `sda1`:

```go
diskio.New("sda1").Output(func(i diskio.IO) bar.Output) {
	return outputs.Textf("sda1: %s", outputs.IByterate(i.Total()))
})
```

## Data: `type IO struct`

### Fields

* `Input unit.Datarate`: Rate of data read from the disk.
* `Output unit.Datarate`: Rate of data written to the disk.

### Methods

* `Total() unit.Datarate`: Total activity of the disk (reads + writes).

[Documentation for unit.Datarate](https://godoc.org/github.com/martinlindhe/unit#Datarate)
