---
title: Timing
---

By default, barista handles the pause/resume from the bar using the `USR1` and `USR2` signals. This
allows suspension of processing while the bar is hidden, saving resources all around.

The `timing` package provides scheduling abilities that are aware of the bar's paused status, and
wait for the bar to resume before firing. This makes them very useful for scheduled, repeating work
that directly results in output, by suspending the work until the bar is visible again.

## Scheduler

`Scheduler`s are pause/resume-aware ticking channels that can be set to trigger at a specific time,
after a specific delay, or repeatedly with a specific interval.

Create a scheduler linked to the bar using `timing.NewScheduler()`.

- `After(time.Duration)`: Tick after the given duration.
- `At(time.Time)`: Tick at a specific time. If in the past, tick immediately.
- `Every(time.Duration)`: Tick after the given duration, and repeat.

`C` holds a `chan <-struct{}` that will be notified for each tick of the scheduler, similar to
[`time.Ticker`](https://golang.org/pkg/time/#Ticker). It is almost guaranteed that the bar is active
when a value is received on this channel. (There's a small chance that the bar has paused again in
the time since the signal was sent, but this is unlikely).

## Example

Typical usage of a scheduler would be in a `for` loop, especially combined with a `select`.

```go
func (m *Module) Stream(sink bar.Sink) {
	sch := timing.NewScheduler().Every(3*time.Second)
	settings := m.settings.Get()
	data := process()
	for {
		sink.Output(settings.format(data))
		select {
			case <-m.settings.Next():
				settings = m.settings.Get()
			case <-sch.C:
				data = process()
		}
	}
}
```

This basically loops forever, waiting for a change in m.settings, or for a scheduler tick. The
`process()` work only happens on scheduler tick, so while the bar is hidden, process will almost
never be called.

## Now

The `timing` package also provides `Now()`, with the same signature as `time.Now()`, but with a few
barista-specific changes:

- When using `timing.TestMode()`, the value returned by `timing.Now()` will be frozen until advanced
  using `NextTick()` or `Advance...()`. The movement of test time is consistent with how schedulers
  are triggered. See the [timing tests](https://github.com/soumya92/barista/blob/master/timing/testmode_test.go)
  for examples.

- The returned `time.Time` will try to be in the machine's local time zone, even if the zone has
  changed since startup. This differs from `time.Now()`, which is pinned to the zone set when the
  binary starts.
