---
title: Gmail
---

Show the number of unread threads in the inbox: `gmail.New(/* client config */)`.  
Show the number of unread threads in one or more labels: `gmail.New(/* client config */, "label1", "other label")`.

Shows the number of unread Gmail messages in the specified labels (defaulting to "INBOX" if none).
It uses the standard [oauth package](/oauth).

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.
* `RefreshInterval(time.Duration)`: Sets the refresh interval. Defaults to 5 minutes.

## Example

<div class="module-example-out">G:0/2/10</div>
<div class="module-example-out">G:4/0/14</div>
Show unread threads in two labels and total threads (read and unread):

```go
gmail.New("INBOX", "other label").Output(func(i gmail.Info) bar.Output {
	return outputs.Textf("G:%d/%d/%d",
		n.Unread["INBOX"], n.Unread["other label"], n.TotalThreads())
})
```

## Data: `type Info struct`

### Fields

* `Unread map[string]int64`: Map of label name to the number of unread threads.
* `Threads map[string]int64`: Map of label name to the total number of threads (including read).

### Methods

* `TotalUnread() int64`: The number of unread threads across all specified labels.
* `TotalThreads() int64`: The total number of threads across all specified labels.

