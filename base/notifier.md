---
title: base/Notifier
---

A notifier can be used to signal changes of interest across goroutines where only the newest value
is relevant and intermediate changed values can be safely ignored.

For example, if a module's output format is changed several times before it gets a chance to update,
it can safely ignore all intermediate values and only use the latest output format. Similarly, if a
scheduler has ticked several times but none of the ticks have been processed, they can be coalesced
into a single tick whenever the consumer is ready next.

`notifier.New()` returns a notification `func()` and a `<-chan struct{}`. Whenever the function is
called, the struct will receive an update, unless the previous update is still pending.

`notifier.Signaller` can be used to signal multiple listeners at once, by providing a
`Next() <-chan struct{}` func that closes the returned chanel when the Signaller is `Signal`ed.

`notifier.SubscribeTo` provides a continuous subscription to signaller-style events, by
automatically re-registering for notifications when the channel is closed. However, because it is
not a one-shot notification, it must be cleaned up using the returned done func.

## Examples

### Notifier
```go
fn, n := notifier.New()

go func(ch <-chan struct{}) {
	for range ch {
		fmt.Println(time.Now())
		time.Sleep(10 * time.Second)
	}
}(n)

for i := 0; i < 10; i++ {
	fn()
	// Don't do this, use timing.NewScheduler() instead.
	time.Sleep(3 * time.Second)
}
```

In this example, even though we sent 10 signals spaced 3 seconds apart, because of blocking delays
in the receiving goroutine, only 4 times will be printed (+0, +10, +20, +30 seconds); the remaining
signals will be discarded instead of being queued up.

### Signaller
```go
s := new(notifier.Signaller)

for i := 0; i < 10; i++ {
	go func(i int) {
		for range s.Next() { fmt.Println(i) }
	}(i)
}

s.Signal() // Will print all 10 almost immediately.

// After the signal, the channels have been closed and cleaned up, so subsequent
// calls will not trigger previous listeners.
s.Signal() // Has no effect.
```

### Subscriber
```go

// Typically in a Stream() func
sub, done := notifier.SubscribeTo(someValue.Next)
defer done()
tick, done := notifier.SubscribeTo(someScheduler.Tick)
defer done()

for {
	select {
		case <-sub:
			doSomethingWith(someValue.Get())
		case <-tick:
			doSomethingPeriodic()
	}
}
```

In this example, we subscribe to two different sources that follow the pattern of returning closing
`<-chan struct{}`s for notifications, and multiplex them. This example also demonstrates the `done`
func returned by SubscribeTo, and how it should typically be deferred for automatic cleanup.
