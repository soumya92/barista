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

`notifier.Source` can be used to signal multiple listeners at once, by providing one-shot listeners
using `Next() <-chan struct{}` that returns a chanel closed on the next `Notify` call, and
`Subscribe() (<-chan struct{}, func())` that returns a channel that receives an empty value on each
notification and an associated cleanup func.

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

### Source
```go
s := new(notifier.Source)

for i := 0; i < 10; i++ {
	go func(i int) {
		for range s.Next() {
			// This won't work, since the channel from Next() never receives a
			// value, it's simply closed.
			fmt.Println(i)
		}
	}(i)

	go func(i int) {
		n := s.Next()
		<-n // This will wait for the next notification.
		fmt.Println(i)
		_, open := <-n
		fmt.Println("channel open: %v", open) // Will print false.
	}(i)
}

go func() {
	sub, done := s.Subscribe()
	defer done()
	for range sub {
		fmt.Println("notified")
	}
}()

s.Notify() // Prints 0-9 and "notifed".

// After the signal, the channels have been closed and cleaned up, so subsequent
// calls will not trigger the 'Next()' listeners. Subscriptions will continue
// to be triggered.
s.Notify() // Only prints "notified", since that's a continuous listener.
```

In this example, we subscribe to a source using both one-shot `Next()` and continuous `Subscribe()`.
This example also demonstrates how the `done` func returned by Subscribe should typically be
deferred for automatic cleanup of subscriptions.
