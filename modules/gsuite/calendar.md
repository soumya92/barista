---
title: Google Calendar
---

Show upcoming events: `calendar.New(/* client config */)`.  

Shows upcoming events on a Google Calendar. It uses the standard [oauth package](/oauth).

## Configuration

* `Output(func(events EventList) (bar.Output, time.Time))`: Sets the output format.
  The returned output is sent to the bar, and the output function is called again at the returned
  Time. This can be especially useful if the output format includes time remaining or elapsed.

* `RefreshInterval(time.Duration)`: Sets the refresh interval. Defaults to 10 minutes.
  This is only the interval at which new events are downloaded. The output can change more
  frequently based on the result of the Output() function.

* `TimeWindow(time.Duration)`: Controls how far in the the future events should be loaded from.
  Defaults to 18 hours.

* `CalendarID(string)`: The calendar ID to fetch events from. Defaults to `"primary"`.

* `ShowDeclined(bool)`: If true, include declined events; If false (default), remove declined events
  from all event lists.

## Example

<div class="module-example-out">Cal: 2h10m</div>
<div class="module-example-out">Cal: -0h5m</div>
Show time until the next event, refreshed every minute:

```go
calendar.New().Output(func(evts calendar.EventList) (bar.Output, time.Time) {
	allEvts := append(append(evts.InProgress, evts.Alerting...), evts.Upcoming...)
	if len(allEvts) == 0 {
		return nil, time.Time{}
	}
	e := allEvts[0]
	untilStart := e.UntilStart()
	minus := ""
	if untilStart < 0 {
		untilStart = -untilStart
		minus = "-"
	}
	return outputs.Textf("Cal: %s%dh%dm",
			minus, int(untilStart.Hours()), int(untilStart.Minutes())%60),
		time.Now().Add(time.Minute)
})
```

## Data: `type EventList struct`

### Fields

* `InProgress []Event`: All events currently in progress.
* `Alerting []Event`: Events where the time until start is less than the
  notification duration. The notification duration is merged from the calendar
  default and any event-specific settings, using the earliest `"popup"`.
* `Upcoming []Event`: All other future events (limited by TimeWindow).

## `type Event struct`

* `Start time.Time`: Start time of the event
* `End time.Time`: End time of the event
* `Alert time.Time`: Time at which a notification should be shown. Same as
  `Start` if no alerts are configured. This is the earliest `"popup"` alert
  configured for the event (or the calendar if no alerts are configured for
  the event specifically).
* `EventStatus Status`: Status of the event (not your response)
* `Response Status`: Your response to the event
* `Location string`: Location of the event
* `Summary string`: Summary of the event

### Methods

* `UntilStart() time.Duration`: Time remaining (or since) the start of the event.
* `UntilEnd() time.Duration`: Time remaining (or since) the end of the event.
* `UntilAlert() time.Duration`: Time remaining (or since) the earliest `"popup"`
  notification for this event.
