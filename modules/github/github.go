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

// Package github provides a barista module to show github notifications.
package github // import "barista.run/modules/github"

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/oauth"
	"barista.run/outputs"
	"barista.run/timing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Notifications represents the notifications grouped by reasons. The key is the
// reason ("comment", "mention"), and the value is the number of notifications
// for that reason. See https://developer.github.com/v3/activity/notifications/#notification-reasons
// for a list of reasons.
type Notifications map[string]int

func (n Notifications) Total() int {
	t := 0
	for _, c := range n {
		t += c
	}
	return t
}

type Module struct {
	config     *oauth.Config
	outputFunc value.Value // of func(Notifications) bar.Output

	// Use the poll interval and last modified from the previous response to
	// control when we next check for notifications.
	scheduler    timing.Scheduler
	lastModified string
}

func New(clientID, clientSecret string) *Module {
	config := oauth.Register(&oauth2.Config{
		Endpoint:     github.Endpoint,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"notifications"},
	})
	m := &Module{
		config:    config,
		scheduler: timing.NewScheduler(),
	}
	m.Output(func(n Notifications) bar.Output {
		if n.Total() == 0 {
			return nil
		}
		return outputs.Textf("GH: %d", n.Total())
	})
	return m
}

type ghNotification struct {
	Reason string
	Unread bool
}

// for tests, to wrap the client in a transport that redirects requests.
var wrapForTest func(*http.Client)

func (m *Module) Stream(sink bar.Sink) {
	client, _ := m.config.Client()
	if wrapForTest != nil {
		wrapForTest(client)
	}
	outf := m.outputFunc.Get().(func(Notifications) bar.Output)
	nextOutputFunc := m.outputFunc.Next()
	info, err := m.getNotifications(client)
	for {
		if err != errCached {
			if sink.Error(err) {
				return
			}
			sink.Output(outf(info))
		}
		err = nil
		select {
		case <-nextOutputFunc:
			nextOutputFunc = m.outputFunc.Next()
			outf = m.outputFunc.Get().(func(Notifications) bar.Output)
		case <-m.scheduler.Tick():
			i, e := m.getNotifications(client)
			err = e
			if e != errCached {
				info = i
			}
		}
	}
}

// This is a terrible hack.
var errCached = errors.New("NothingChanged")

func (m *Module) getNotifications(client *http.Client) (Notifications, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/notifications", nil)
	if m.lastModified != "" {
		req.Header.Add("If-Modified-Since", m.lastModified)
	}
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	m.lastModified = r.Header.Get("Last-Modified")
	interval, _ := strconv.ParseInt(r.Header.Get("X-Poll-Interval"), 10, 64)
	if interval < 10 {
		interval = 10
	}
	m.scheduler.After(time.Duration(interval) * time.Second)
	if r.StatusCode == 304 {
		return nil, errCached
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP Status %d", r.StatusCode)
	}
	info := Notifications{}
	resp := []ghNotification{}
	err = json.NewDecoder(r.Body).Decode(&resp)
	defer r.Body.Close()
	if err != nil {
		return nil, err
	}
	for _, n := range resp {
		if !n.Unread {
			continue
		}
		count := info[n.Reason]
		count++
		info[n.Reason] = count
	}
	return info, nil
}

func (m *Module) Output(outputFunc func(Notifications) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}
