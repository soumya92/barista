---
title: base/Value
---

`Value` and `ErrorValue` provide atomic value storage with update notifications. Mostly
used in modules to store configuration and `select {}` changes to the configuration. For example,
most modules will update the output when either the interval elapses, or the output function is
changed, so a common pattern is:

```go

type Module struct {
	outputFormat value.Value
	// ...
}

// ...

func (m *Module) Stream(sink bar.Sink) {
	// ...
	for {
		select {
			case <-m.outputFormat.Next():
				format = m.outputFormat.Get().(/* output format type */)
			case <-scheduler.Tick():
				data = /* read new data */
		}
		// output with format, data
	}
}
```

## Value

`Value` provides simple atomic storage and update notifications:

* `Get() interface{}`: Gets the previously stored value, or `nil` if nothing was stored.

* `Set(interface{})`: Stores a new value and notifies any listeners of the update.

* `Next() <-chan struct{}`: Listens for the next value update. Each call to `Next()` creates a new
  channel that will be closed the next time the value changes, so once the channel notifies it is
  no longer useful, and `Next()` should be called again for the next change of interest.

## ErrorValue

`ErrorValue` extends Value with support for storing errors.

* `Get() (interface{}, error)`: Gets the previously stored value or error. At most one will be set,
  but both can be `nil` if no value or error has been stored before.

* `Set(interface{})`: Sets a non-error value, clearing any previous error.

* `Error(error) bool`: If given a nil error, returns false and does not change the value. Otherwise
  clears any previously set value and replaces it with the error. This is similar to `bar.Sink#Error`,
  allowing code like:

  ```go
err, raw := /* something */
if ev.Error(err) {
	return
}
data := process(raw)
ev.Set(data)
```

* `Next() <-chan struct{}`: Same as `Value`.
