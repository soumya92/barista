---
title: Function Modules
---

Package `funcs` facilitates creating simple modules using nothing more than a go function.

This is intended only for creating simple, specialized modules that are not useful in general.
To create a shareable module, see the guide for [Writing a Custom Module](/docs/writing-a-module).

## Once

The simplest of all modules, just executes the given function and sends the output to the bar.
Because the function is passed a [`bar.Sink`](/bar#Sink), it can update the output multiple times
(e.g. to indicate progress loading something) before finally finishing with the output that will
stay on the bar forever.

### Example

<div class="module-example-out">Liftoff!</div>
Counting down 5 seconds, and then staying at "Liftoff!":

```go
funcs.Once(func(s bar.Sink) {
	for i := 5; i > 0; i-- {
		s.Output(outputs.Textf("%d...", i))
		time.Sleep(time.Second)
	}
	s.Output("Liftoff!")
})
```

## OnClick

Very similar to a module created with `Once`, the only difference is that once the function returns,
(Left/Right/Middle) clicking on the output will restart the module and call the function again.

## Every

Repeatedly calls the given function at a fixed interval. The timer is independent of the function,
so if the function runs longer than the interval, it will be called again immediately. However,
updates do not accumulate, so it will only be called once for any number of intervals spanned.

### Example

<div class="module-example-out">5 unread</div>
Getting the number of unread messages using the [Gmail API](https://developers.google.com/gmail/api/quickstart/go):

```go
srv, _ := gmail.New(client)

gmailModule := funcs.Every(5*time.Minute, func(s bar.Sink) {
	s.Output(outputs.Text("..."))
	r, err := srv.Users.Labels.Get("me", "INBOX").Do()
	if !s.Error(err) {
		s.Output(outputs.Textf("%d unread", r.messagesUnread))
	}
})
```
