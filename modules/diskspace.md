---
title: Disk Utilisation
---

Display disk utilisation stats for a path: `diskspace.New("/home/me")` / `diskspace.New("/mnt/sdcard")`.

Note: The path given to diskspace should be a path *on* the device (e.g. `/mnt/sdcard`), and not the
path to the block device (e.g. `/dev/mmcblk0`).

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.

* `RefreshInterval(time.Duration)`: Sets the interval to wait before fetching updated disk
  utilisation information. Defaults to 3 seconds.

## Example

<div class="module-example-out">144.8 GiB avail</div>
Show the space available on the home folder's disk:

```go
diskspace.New("/home").Output(func(i diskspace.Info) bar.Output) {
	return outputs.Textf("%s avail", outputs.IBytesize(i.Available))
})
```

## Data: `type Info struct`

### Fields

* `Available unit.Datasize`: Available disk space (for non-root user).
* `Free unit.Datasize`: Free disk space, usually larger than Available because some space is reserved for root.
* `Total unit.Datasize`: Total disk capacity.

### Methods

* `Used() unit.Datasize`: disk space currently in use.
* `UsedFrac() float64`: the fraction of disk space currently in use.
* `UsedPct() int`: the percentage of disk space currently in use.
* `AvailFrac() float64`: the fraction of disk space available.
* `AvailPct() int`: the percentage of disk space available.

[Documentation for unit.Datasize](https://godoc.org/github.com/martinlindhe/unit#Datasize)
