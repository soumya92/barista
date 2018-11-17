// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package calendar provides a Google Calendar barista module.
package calendar // import "barista.run/modules/gsuite/calendar"

import (
	"net/http"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/oauth"
	"barista.run/outputs"
	"barista.run/timing"

	"golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
)

// Status represents the response/status for an event.
type Status string

// Possible values for status per the calendar API.
const (
	StatusUnknown     = Status("")
	StatusConfirmed   = Status("confirmed")
	StatusTentative   = Status("tentative")
	StatusCancelled   = Status("cancelled")
	StatusDeclined    = Status("declined")
	StatusUnresponded = Status("needsAction")
)

// Event represents a calendar event.
type Event struct {
	Start       time.Time
	End         time.Time
	Alert       time.Time
	EventStatus Status
	Response    Status
	Location    string
	Summary     string
}

// UntilStart returns the time remaining until the event starts.
func (e Event) UntilStart() time.Duration {
	return e.Start.Sub(timing.Now())
}

// UntilEnd returns the time remaining until the event ends.
func (e Event) UntilEnd() time.Duration {
	return e.End.Sub(timing.Now())
}

// UntilAlert returns the time remaining until a notification should be
// displayed for this event.
func (e Event) UntilAlert() time.Duration {
	return e.Alert.Sub(timing.Now())
}

// EventList represents the list of events split by the temporal state of each
// event: in progress, alerting (upcoming but within notification duration), or
// upcoming beyond the notification duration.
type EventList struct {
	// All events currently in progress
	InProgress []Event
	// Events where the time until start is less than the notification duration
	Alerting []Event
	// All other future events
	Upcoming []Event
}

type config struct {
	calendarID   string
	lookahead    time.Duration
	showDeclined bool
}

// Module represents a Google Calendar barista module.
type Module struct {
	oauthConfig *oauth.Config
	config      value.Value // of config
	scheduler   *timing.Scheduler
	outputFunc  value.Value // of func(EventList) bar.Output
}

// New creates a calendar module from the given oauth config.
func New(clientConfig []byte) *Module {
	conf, err := google.ConfigFromJSON(clientConfig, calendar.CalendarReadonlyScope)
	if err != nil {
		panic("Bad client config: " + err.Error())
	}
	m := &Module{
		oauthConfig: oauth.Register(conf),
		scheduler:   timing.NewScheduler(),
	}
	m.config.Set(config{calendarID: "primary"})
	m.RefreshInterval(10 * time.Minute)
	m.TimeWindow(18 * time.Hour)
	m.Output(func(evts EventList) bar.Output {
		hasEvent := false
		out := outputs.Group()
		for _, e := range evts.InProgress {
			out.Append(outputs.Textf("ends %s: %s",
				e.End.Format("15:04"), e.Summary))
			hasEvent = true
		}
		for _, e := range evts.Alerting {
			out.Append(outputs.Textf("%s: %s",
				e.Start.Format("15:04"), e.Summary))
			hasEvent = true
		}
		if !hasEvent {
			for _, e := range evts.Upcoming {
				// If no other events have been displayed, show next upcoming.
				out.Append(outputs.Textf("%s: %s",
					e.Start.Format("15:04"), e.Summary))
				break
			}
		}
		return out
	})
	return m
}

// for tests, to wrap the client in a transport that redirects requests.
var wrapForTest func(*http.Client)

// Stream starts the module.
func (m *Module) Stream(sink bar.Sink) {
	client, _ := m.oauthConfig.Client()
	if wrapForTest != nil {
		wrapForTest(client)
	}
	srv, _ := calendar.New(client)
	outf := m.outputFunc.Get().(func(EventList) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()
	conf := m.getConfig()
	nextConfig, done := m.config.Subscribe()
	defer done()
	renderer := timing.NewScheduler()
	evts, err := fetch(srv, conf)
	for {
		if sink.Error(err) {
			return
		}
		list, refresh := makeEventList(evts)
		if !refresh.IsZero() {
			renderer.At(refresh.Add(time.Duration(1)))
		}
		sink.Output(outf(list))
		select {
		case <-nextOutputFunc:
			outf = m.outputFunc.Get().(func(EventList) bar.Output)
		case <-nextConfig:
			conf = m.getConfig()
			evts, err = fetch(srv, conf)
		case <-m.scheduler.C:
			evts, err = fetch(srv, conf)
		case <-renderer.C:
		}
	}
}

func fetch(srv *calendar.Service, conf config) ([]Event, error) {
	timeMin := timing.Now()
	timeMax := timeMin.Add(conf.lookahead)

	req := srv.Events.List(conf.calendarID)
	req.MaxAttendees(1)
	req.OrderBy("startTime")
	// Simplify recurring events by converting them to single events.
	req.SingleEvents(true)
	req.TimeMin(timeMin.Format(time.RFC3339))
	req.TimeMax(timeMax.Format(time.RFC3339))
	req.Fields("items(end,location,start,status,summary,attendees,reminders),defaultReminders")
	res, err := req.Do()
	if err != nil {
		return nil, err
	}
	defaultAlert := getEarliestPopupReminder(res.DefaultReminders)
	events := []Event{}
	for _, e := range res.Items {
		if e.Start.DateTime == "" || e.End.DateTime == "" {
			// All day events only have .Date, not .DateTime.
			continue
		}
		start, err := time.Parse(time.RFC3339, e.Start.DateTime)
		if err != nil {
			return nil, err
		}
		end, err := time.Parse(time.RFC3339, e.End.DateTime)
		if err != nil {
			return nil, err
		}
		if end.Before(timeMin) {
			continue
		}
		selfStatus := StatusUnknown
		for _, at := range e.Attendees {
			if at.Self {
				selfStatus = Status(at.ResponseStatus)
			}
		}
		if !conf.showDeclined && selfStatus == StatusDeclined {
			continue
		}
		alert := defaultAlert
		if e.Reminders != nil && !e.Reminders.UseDefault {
			alert = getEarliestPopupReminder(e.Reminders.Overrides)
		}
		eventStatus := Status(e.Status)
		if eventStatus == StatusCancelled {
			continue
		}
		events = append(events, Event{
			Start:       start,
			End:         end,
			Alert:       start.Add(-alert),
			EventStatus: eventStatus,
			Response:    selfStatus,
			Location:    e.Location,
			Summary:     e.Summary,
		})
	}
	return events, nil
}

func getEarliestPopupReminder(rs []*calendar.EventReminder) time.Duration {
	duration := time.Duration(0)
	for _, r := range rs {
		if r.Method != "popup" {
			continue
		}
		rDuration := time.Duration(r.Minutes) * time.Minute
		if rDuration > duration {
			duration = rDuration
		}
	}
	return duration
}

func makeEventList(events []Event) (EventList, time.Time) {
	now := timing.Now()
	var refresh time.Time
	list := EventList{}
	for _, e := range events {
		switch {
		case now.After(e.End):
			continue
		case now.After(e.Start) && e.End.After(now):
			setIfEarlier(&refresh, e.End)
			list.InProgress = append(list.InProgress, e)
		case now.After(e.Alert) && e.Start.After(now):
			setIfEarlier(&refresh, e.Start)
			list.Alerting = append(list.Alerting, e)
		default:
			setIfEarlier(&refresh, e.Alert)
			list.Upcoming = append(list.Upcoming, e)
		}
	}
	return list, refresh
}

func setIfEarlier(target *time.Time, source time.Time) {
	if target.IsZero() || source.Before(*target) {
		*target = source
	}
}

// Output sets the output format for the module.
func (m *Module) Output(outputFunc func(EventList) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval sets the interval for fetching new events. Note that this is
// distinct from the rendering interval, which is returned by the output func
// on each new output.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

func (m *Module) getConfig() config {
	return m.config.Get().(config)
}

// CalendarID sets the ID of the calendar to fetch events for.
func (m *Module) CalendarID(id string) *Module {
	c := m.getConfig()
	c.calendarID = id
	m.config.Set(c)
	return m
}

// TimeWindow controls the search window for future events.
func (m *Module) TimeWindow(window time.Duration) *Module {
	c := m.getConfig()
	c.lookahead = window
	m.config.Set(c)
	return m
}

// ShowDeclined controls whether declined events are shown or ignored.
func (m *Module) ShowDeclined(show bool) *Module {
	c := m.getConfig()
	c.showDeclined = show
	m.config.Set(c)
	return m
}
