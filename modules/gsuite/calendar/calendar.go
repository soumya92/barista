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

// InProgress returns true if the event is currently in progress.
func (e Event) InProgress() bool {
	now := timing.Now()
	return !e.Start.After(now) && e.End.After(now)
}

// Finished returns true if the event is over.
func (e Event) Finished() bool {
	return !e.End.After(timing.Now())
}

type config struct {
	calendar     string
	lookahead    time.Duration
	lookbehind   time.Duration
	showDeclined bool
}

type Module struct {
	oauthConfig *oauth.Config
	config      value.Value // of config
	scheduler   timing.Scheduler
	outputFunc  value.Value // of func(*Event, *Event) (bar.Output, time.Duration)
}

func New(clientConfig []byte) *Module {
	conf, err := google.ConfigFromJSON(clientConfig, calendar.CalendarReadonlyScope)
	if err != nil {
		panic("Bad client config: " + err.Error())
	}
	m := &Module{
		oauthConfig: oauth.Register(conf),
		scheduler:   timing.NewScheduler(),
	}
	m.config.Set(config{calendar: "primary"})
	m.RefreshInterval(10 * time.Minute)
	m.TimeWindow(5*time.Minute, 18*time.Hour)
	m.Output(func(e, next *Event) (bar.Output, time.Duration) {
		const alertTime = -5 * time.Minute
		if e == nil || e.Finished() {
			// If e is nil or finished, next will not be defined.
			return nil, 0
		}
		refresh := e.UntilEnd()
		if next != nil && next.Start.Add(alertTime).Before(e.End) {
			refresh = next.UntilStart() + alertTime
		}
		if e.InProgress() {
			return outputs.Textf("ends %s: %s", e.End.Format("15:04"), e.Summary), refresh
		}
		out := outputs.Textf("%s: %s", e.Start.Format("15:04"), e.Summary)
		refresh = e.UntilStart()
		if timing.Now().After(e.Start.Add(alertTime)) {
			return out.Urgent(true), refresh
		}
		return out, refresh + alertTime
	})
	return m
}

// for tests, to wrap the client in a transport that redirects requests.
var wrapForTest func(*http.Client)

func (m *Module) Stream(sink bar.Sink) {
	client, _ := m.oauthConfig.Client()
	if wrapForTest != nil {
		wrapForTest(client)
	}
	srv, err := calendar.New(client)
	if sink.Error(err) {
		return
	}
	outf := m.outputFunc.Get().(func(*Event, *Event) (bar.Output, time.Duration))
	nextOutputFunc := m.outputFunc.Next()
	conf := m.getConfig()
	nextConfig := m.config.Next()
	renderer := timing.NewScheduler()
	evts, err := fetch(srv, conf)
	for {
		if sink.Error(err) {
			return
		}
		evt, idx := getCurrentEvent(evts, conf)
		var nextEvt *Event
		if idx > 0 && idx < len(evts) {
			nextEvt, _ = getCurrentEvent(evts[idx:], conf)
		}
		out, refresh := outf(evt, nextEvt)
		if refresh > 0 {
			refresh++
			renderer.After(refresh)
		}
		sink.Output(out)
		select {
		case <-nextOutputFunc:
			nextOutputFunc = m.outputFunc.Next()
			outf = m.outputFunc.Get().(func(*Event, *Event) (bar.Output, time.Duration))
		case <-nextConfig:
			nextConfig = m.config.Next()
			conf = m.getConfig()
			evts, err = fetch(srv, conf)
		case <-m.scheduler.Tick():
			evts, err = fetch(srv, conf)
		case <-renderer.Tick():
		}
	}
}

func fetch(srv *calendar.Service, conf config) ([]Event, error) {
	timeMin := timing.Now().Add(-conf.lookbehind)
	timeMax := timing.Now().Add(conf.lookahead)

	req := srv.Events.List(conf.calendar)
	req.MaxAttendees(1)
	req.MaxResults(15)
	req.OrderBy("startTime")
	// Simplify recurring events by converting them to single events.
	req.SingleEvents(true)
	req.TimeMax(timeMax.Format(time.RFC3339))
	req.TimeMin(timeMin.Format(time.RFC3339))
	req.Fields("items(end,location,start,status,summary,attendees,reminders)")
	res, err := req.Do()
	if err != nil {
		return nil, err
	}
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
		eventStatus := Status(e.Status)
		events = append(events, Event{
			Start:       start,
			End:         end,
			EventStatus: eventStatus,
			Response:    selfStatus,
			Location:    e.Location,
			Summary:     e.Summary,
		})
	}
	return events, nil
}

func getCurrentEvent(events []Event, conf config) (*Event, int) {
	now := timing.Now()
	for i, e := range events {
		if e.End.Before(now) {
			continue
		}
		if e.EventStatus == StatusCancelled || e.Response == StatusCancelled {
			continue
		}
		if e.Response == StatusDeclined && !conf.showDeclined {
			continue
		}
		return &e, i + 1
	}
	// Didn't find any event.
	return nil, -1
}

func (m *Module) Output(outputFunc func(*Event, *Event) (bar.Output, time.Duration)) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}

func (m *Module) getConfig() config {
	return m.config.Get().(config)
}

func (m *Module) CalendarID(id string) *Module {
	c := m.getConfig()
	c.calendar = id
	m.config.Set(c)
	return m
}

func (m *Module) TimeWindow(past, future time.Duration) *Module {
	c := m.getConfig()
	c.lookbehind = past
	c.lookahead = future
	m.config.Set(c)
	return m
}

func (m *Module) ShowDeclined(show bool) *Module {
	c := m.getConfig()
	c.showDeclined = show
	m.config.Set(c)
	return m
}
