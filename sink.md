---
title: Sink
---

The `sink` package provides methods to construct different kinds of sinks, especially useful for
testing or wrapping existing modules.

- `sink.Func(func(bar.Segments))`

  Returns a sink that converts outputs into `bar.Segments` before passing them along to the given
  func. If more fine-grained control is required (such as detecting runtime extensions of
  `bar.Output`) use a `func(bar.Output)` as a `bar.Sink` directly.

- `sink.New()`

  Returns a `<-chan bar.Segments` and a linked `bar.Sink`. Any values sent to the returned sink will
  be sent to the channel, and the sink will block until the value has been read.

- `sink.Buffered(int)`

  Like `sink.New()`, but the channel has a buffer of the given size.

- `sink.Null()`

  Returns a sink that discards all output sent to it.

- `sink.Value()`

  Returns a sink that stores the latest output in a [base/value](/base/value), as `bar.Segments`.
  Call `.Get().(bar.Segments)` to get the latest output.

## Examples

```go
ch, sink := sink.New()
go someModule.Stream(sink)

for out := range ch {
	// out will contain each output from someModule.
}
```

Using a value sink:

```go
val, sink := sink.New()
go someModule.Stream(sink)

sub, done := val.Subscribe()
defer done()

for range sub {
	out := val.Get().(bar.Segments)
	// out will contain the latest output from the module.
	// Because of coalescing, some outputs may be dropped if a new output is received before the
	// previous output was completely processed.
}
```
