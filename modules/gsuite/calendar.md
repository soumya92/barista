---
title: Google Calendar
---

Show upcoming events: `calendar.New(/* client config */)`.  

Shows upcoming events on a Google Calendar. It uses the standard [oauth package](/oauth).

## Configuration

* `Output(func(current, next *Event) (bar.Output, time.Duration))`: Sets the output format.
  The current event (can be nil), and the next event (can be nil) are provided for some flexibility.
  The returned output is sent to the bar, and the output function is called again after the duration
  returned. This can be especially useful if the output format includes time remaining or elapsed.

* `RefreshInterval(time.Duration)`: Sets the refresh interval. Defaults to 10 minutes.
  This is only the interval at which new events are downloaded. The output can change more
  frequently based on the result of the Output() function.

* `TimeWindow(past, future time.Duration)`: Used as API parameters, indicates how far in the past
  and into the future events should be loaded from. Defaults to 5 minutes in the past and 18 hours
  in the future.

* `CalendarID(string)`: The calendar ID to fetch events from. Defaults to `"primary"`.

* `ShowDeclined(bool)`: If true, treat declined events normally. If false (default), skip over them.

## Example

<div class="module-example-out">Cal: 2h10m</div>
<div class="module-example-out">Cal: -0h5m</div>
Show time until the next event, refreshed every minute:

```go
calendar.New().Output(func(e, n *calendar.Event) (bar.Output, time.Duration) {
	if e == nil {
		return nil, 0
	}
	untilStart := e.UntilStart()
	minus := ""
	if untilStart < 0 {
		untilStart = -untilStart
		minus = "-"
	}
	return outputs.Textf("Cal: %s%dh%dm",
			minus, int(untilStart.Hours()), int(untilStart.Minutes())%60),
		time.Minute
})
```

## Data: `type Event struct`

### Fields

* `Start time.Time`: Start time of the event
* `End time.Time`: End time of the event
* `EventStatus Status`: Status of the event (not your response)
* `Response Status`: Your response to the event
* `Location string`: Location of the event
* `Summary string`: Summary of the event

### Methods

* `UntilStart() time.Duration`: Time remaining (or since) the start of the event.
* `UntilEnd() time.Duration`: Time remaining (or since) the end of the event.
* `InProgress() bool`: True if the event is currently in progress.
* `Finished() bool`: True if the event is finished.
