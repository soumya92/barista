---
title: Formatting
---

The `format` package provides some utility methods for formatting data units.

- `format.Bytesize(unit.Datasize)` formats the given datasize using [go-humanize](https://godoc.org/github.com/dustin/go-humanize).
- `format.IBytesize(unit.Datasize)` formats the datasize in IEC units.

- `format.Byterate(unit.Datarate)` formats the given datarate by appending "/s" to the formatted bytesize.
- `format.IByterate(unit.Datarate)` formats the given datarate in IEC units.
