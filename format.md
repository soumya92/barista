---
title: Formatting
---

The `format` package provides some utility methods for formatting units. Each of
the formatting functions returns a `format.Value`, where the numeric value is
can be obtained in a format the fits a specific width (up to 8), and the unit is
prefixed according to the magnitude of the value (`k`, `M`, `µ`, &c.).

This value can be passed directly to `outputs.Pango` or `pango.Unit` to be
formatted according to the default configuration set for the bar.

- `format.Bytesize(unit.Datasize)` formats the given datasize using [go-humanize](https://godoc.org/github.com/dustin/go-humanize).
- `format.IBytesize(unit.Datasize)` formats the datasize in IEC units.

- `format.Byterate(unit.Datarate)` formats the given datarate by appending "/s" to the formatted bytesize.
- `format.IByterate(unit.Datarate)` formats the given datarate in IEC units.
