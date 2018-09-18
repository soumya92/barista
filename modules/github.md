---
title: GitHub Notifications
---

Show the number of unread GitHub notifications: `github.New()`.  
Using a custom Client ID and Secret: `github.NewWithClientID("id", "secret")`.

Shows the number of unread GitHub Notifications, optionally broken down by reason.
(See the [list of reasons](https://developer.github.com/v3/activity/notifications/#notification-reasons)).
It uses the standard [oauth package](/oauth). However, GitHub does not provide easy
offline authentication, so you need to manually extract the token from the success
URL. After completing oauth, you will end up on the page `https://github.com/login/oauth/success?code=xxxx`.
The string after `?code=` is what needs to be pasted in the interactive oauth setup.

## Configuration

* `Output(func(Notifications) bar.Output)`: Sets the output format.

The refresh interval is automatically set using the `X-Poll-Interval` header.

## Example

<div class="module-example-out">GH:4</div>
<div class="module-example-out">Mentions:2</div>
Show mentions urgently, otherwise all unread notifications:

```go
github.New().Output(func(n github.Notifications) bar.Output {
	if n["mention"] > 0 {
		return outputs.Textf("Mentions:%d", n["mention"]).Urgent(true)
	}
	if n.Total() == 0 {
		return nil
	}
	return outputs.Textf("GH:%d", n.Total())
})
```

## Data: `type Notifications map[string]int`

Keys are reasons, e.g. `"mention"`, `"assign"`, and values are the number of unread
notifications with that reason.

### Methods

* `Total() int`: The total number of unread notifications.

