---
title: Mock I/O
---

The `mockio` testing package provides in-memory streams that allow assertions against content read
and written from them.


## Readable

Create a new Readable using `mockio.Stdin()`.

`mockio.Readable` is a readable stream, simulating stdin. It also implements `io.Writer`, so it can
be written to using the standard I/O utilities. It can simulate an error on the next read using
`ShouldError(error)`.


## Writable

Create a new Writable using `mockio.Stdout()`.

`mockio.Writable` is a writable stream, simulating stdout. It provides a few different methods to
read the data written to it:

- `ReadNow() string`: Returns a string with all content written since the last read.
- `ReadUntil(delim byte, timeout time.Duration) (string, error)`: Reads until the next occurrence of
  the delimiting byte, or timeout, whichever occurs first. Returns the contents as a string, and an
  error if timeout expired first.
- `WaitForWrite(time.Duration) bool`: Returns true if anything was written before the timeout. It
  does not read, so the next call to ReadNow/ReadUntil will return the content.

It also provides `ShouldError(error)` to simulate an error on next write.
