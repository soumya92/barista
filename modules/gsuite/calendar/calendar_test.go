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

package calendar

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	"barista.run/testing/httpclient"
	"barista.run/timing"
	"github.com/stretchr/testify/require"
)

type datetime struct {
	Date     string `json:"date,omitempty"`
	DateTime string `json:"datetime,omitempty"`
}

type attendee struct {
	Self           bool   `json:"self"`
	ResponseStatus string `json:"responseStatus"`
}

type reminder struct {
	Method  string `json:"method"`
	Minutes int    `json:"minutes"`
}

type reminders struct {
	UseDefault bool       `json:"useDefault"`
	Reminders  []reminder `json:"overrides"`
}

// test events mapped more to the calendar api.
type event struct {
	Start       datetime   `json:"start"`
	End         datetime   `json:"end"`
	EventStatus string     `json:"status"`
	Attendees   []attendee `json:"attendees"`
	Location    string     `json:"location"`
	Summary     string     `json:"summary"`
	Reminders   reminders  `json:"reminders"`
}

var (
	events   = map[string][]event{}
	eventsMu sync.Mutex
)

func setEvents(calendar string, testEvents ...event) {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	events[calendar] = testEvents
}

func resetEvents() {
	eventsMu.Lock()
	defer eventsMu.Unlock()
	events = map[string][]event{}
}

var fakeClientConfig = []byte(`{
	"installed": {
		"client_id": "143832941570-ek4civ0n1csaahcspkpag91dmfmudd7k.apps.googleusercontent.com",
		"project_id": "i3-barista",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://www.googleapis.com/oauth2/v3/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_secret": "yFSEf5c-vgzzDnfb4vLHqAlr",
		"redirect_uris": ["urn:ietf:wg:oauth:2.0:oob", "http://localhost"]
	}
}`)

func TestEmpty(t *testing.T) {
	resetEvents()
	setEvents("primary" /*, no events */)

	testBar.New(t)
	cal := New(fakeClientConfig)
	testBar.Run(cal)
	testBar.NextOutput().AssertEmpty("With no events")
}

func TestCalendar(t *testing.T) {
	fixedTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

	resetEvents()
	e0 := event{Summary: "All-day event"}
	e0.Start.Date = "2000-01-01"
	e0.End.Date = "2000-01-04"

	e1 := event{Summary: "Test event"}
	e1.Start.DateTime = fixedTime.Format(time.RFC3339)
	e1.End.DateTime = fixedTime.Add(15 * time.Minute).Format(time.RFC3339)
	e1.Reminders.UseDefault = true

	newTime := fixedTime.Add(45 * time.Minute)
	e2 := event{Summary: "Declined event",
		Attendees: []attendee{{false, "needsAction"}, {true, "declined"}}}
	e2.Start.DateTime = newTime.Format(time.RFC3339)
	e2.End.DateTime = newTime.Add(45 * time.Minute).Format(time.RFC3339)
	e2.Reminders.UseDefault = true

	newTime = fixedTime.Add(time.Hour)
	e3 := event{Summary: "Cancelled event", EventStatus: "cancelled"}
	e3.Start.DateTime = newTime.Format(time.RFC3339)
	e3.End.DateTime = newTime.Add(15 * time.Minute).Format(time.RFC3339)
	e3.Reminders.UseDefault = true

	newTime = fixedTime.Add(90 * time.Minute)
	e4 := event{Summary: "Instant"}
	e4.Start.DateTime = newTime.Format(time.RFC3339)
	e4.End.DateTime = newTime.Format(time.RFC3339)
	e4.Reminders.UseDefault = true

	newTime = fixedTime.Add(2 * time.Hour)
	e5 := event{Summary: "Later event"}
	e5.Start.DateTime = newTime.Format(time.RFC3339)
	e5.End.DateTime = newTime.Add(15 * time.Minute).Format(time.RFC3339)
	e5.Reminders.UseDefault = false
	e5.Reminders.Reminders = []reminder{{"email", 40}, {"popup", 4}, {"popup", 8}}

	setEvents("primary", e0, e1, e2, e3, e4, e5)

	testBar.New(t)
	timing.AdvanceTo(fixedTime.Add(-15 * time.Minute))

	cal := New(fakeClientConfig).TimeWindow(24 * time.Hour)
	testBar.Run(cal)

	testBar.NextOutput().AssertText([]string{"00:00: Test event"})
	cal.RefreshInterval(720 * time.Hour) // Only test the rendering interval.

	testBar.AssertNoOutput("on refresh interval change")

	newTime = timing.NextTick()
	require.Equal(t, "23:55", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{"00:00: Test event"})

	newTime = timing.NextTick()
	require.Equal(t, "00:00", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{"ends 00:15: Test event"})

	newTime = timing.NextTick()
	require.Equal(t, "00:15", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{"01:30: Instant"})

	newTime = timing.NextTick()
	require.Equal(t, "01:25", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{"01:30: Instant"})

	cal.ShowDeclined(true)
	testBar.NextOutput().AssertText(
		[]string{"ends 01:30: Declined event", "01:30: Instant"},
		"On configuration change")

	// Ensures that a fetch is not performed, since all events will disappear
	// on next fetch.
	setEvents("primary")
	cal.Output(func(l EventList) bar.Output {
		formatEvt := func(e Event) bar.Output {
			if e.UntilStart() < 0 && e.UntilEnd() > 0 {
				return outputs.Textf("until %s: %s",
					e.End.Format("15:04"), e.Summary)
			}
			if e.UntilAlert() < 0 {
				return outputs.Textf("at %s: %s",
					e.Start.Format("15:04"), e.Summary)
			}
			return nil
		}
		switch {
		case len(l.InProgress) > 0:
			return formatEvt(l.InProgress[0])
		case len(l.Alerting) > 0:
			return formatEvt(l.Alerting[0])
		case len(l.Upcoming) > 0:
			return formatEvt(l.Upcoming[0])
		}
		return nil
	})
	testBar.NextOutput().AssertText([]string{"until 01:30: Declined event"},
		"On output format change")

	newTime = timing.NextTick()
	require.Equal(t, "01:30", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{})

	newTime = timing.NextTick()
	require.Equal(t, "01:52", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{"at 02:00: Later event"})

	newTime = timing.NextTick()
	require.Equal(t, "02:00", newTime.Format("15:04"))
	testBar.NextOutput().AssertText([]string{"until 02:15: Later event"})

	newTime = timing.NextTick()
	require.Equal(t, "02:15", newTime.Format("15:04"))
	testBar.NextOutput().AssertEmpty("no more events")

	pastEvent := event{Summary: "past event"}
	pastEvent.Start.DateTime = newTime.Format(time.RFC3339)
	pastEvent.End.DateTime = newTime.Add(time.Hour).Format(time.RFC3339)
	setEvents("primary", pastEvent)

	newTime = timing.NextTick()
	require.Equal(t, "Jan 30", newTime.Format("Jan 2"),
		"Next tick only on refresh interval")
	testBar.NextOutput().AssertEmpty("all events in the past")
}

func TestErrors(t *testing.T) {
	resetEvents()

	require.Panics(t, func() { New([]byte(`not-a-json-config`)) })

	testBar.New(t)
	cal := New(fakeClientConfig).CalendarID("no-such-calendar")
	testBar.Run(cal)
	testBar.NextOutput().AssertError("Calendar ID not found")

	setEvents("primary",
		event{Start: datetime{DateTime: "bad"}, End: datetime{DateTime: "bad"}})

	testBar.New(t)
	cal = New(fakeClientConfig)
	testBar.Run(cal)
	testBar.NextOutput().AssertError("With bad start time")

	now := timing.Now()
	setEvents("primary", event{
		Start: datetime{DateTime: now.Format(time.RFC3339)},
		End:   datetime{DateTime: "bad"},
	})

	testBar.New(t)
	cal = New(fakeClientConfig)
	testBar.Run(cal)
	testBar.NextOutput().AssertError("With bad start time")
}

func TestMain(m *testing.M) {
	mux := http.NewServeMux()
	mux.HandleFunc("/calendar/v3/calendars/", func(w http.ResponseWriter, r *http.Request) {
		cal := strings.Split(r.URL.Path, "/")[4]
		eventsMu.Lock()
		defer eventsMu.Unlock()
		calEvents, ok := events[cal]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var evts struct {
			Reminders []reminder `json:"defaultReminders"`
			Items     []event    `json:"items"`
		}
		for _, e := range calEvents {
			evts.Items = append(evts.Items, e)
		}
		evts.Reminders = append(evts.Reminders,
			reminder{"email", 60},
			reminder{"popup", 1},
			reminder{"popup", 3},
			reminder{"popup", 5},
		)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(evts)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	wrapForTest = func(c *http.Client) {
		httpclient.FreezeOauthToken(c, "authtoken-placeholder")
		httpclient.Wrap(c, server.URL)
	}

	os.Exit(m.Run())
}
