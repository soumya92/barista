---
title: base/Sink
---

The `sink` package provides methods to construct different kinds of sinks, especially useful for
testing or wrapping existing modules.

- `sink.New()`

  Returns a `<-chan bar.Output` and a linked `bar.Sink`. Any values sent to the returned sink will
  be sent to the channel, and the sink will block until the value has been read.

- `sink.Buffered(int)`

  Like `sink.New()`, but the channel has a buffer of the given size.

- `sink.Null()`

  Returns a sink that discards all output sent to it.

- `sink.Value()`

  Returns a sink that stores the latest output in a [base/value](/base/value).

## Example

```go
ch, sink := sink.New()
go someModule.Stream(sink)

for out := range ch {
	// out will contain each output from someModule.
}
```
