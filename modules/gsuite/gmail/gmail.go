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

// Package gmail provides a gmail barista module.
package gmail // import "barista.run/modules/gsuite/gmail"

import (
	"net/http"
	"time"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/oauth"
	"barista.run/outputs"
	"barista.run/timing"

	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
)

type Info struct {
	Threads map[string]int64
	Unread  map[string]int64
}

func (i Info) TotalUnread() int64 {
	t := int64(0)
	for _, u := range i.Unread {
		t += u
	}
	return t
}

func (i Info) TotalThreads() int64 {
	t := int64(0)
	for _, c := range i.Threads {
		t += c
	}
	return t
}

type Module struct {
	config     *oauth.Config
	labels     []string
	scheduler  timing.Scheduler
	outputFunc value.Value // of func(Info) bar.Output
}

func New(clientConfig []byte, labels ...string) *Module {
	config, err := google.ConfigFromJSON(clientConfig, gmail.GmailLabelsScope)
	if err != nil {
		panic("Bad client config: " + err.Error())
	}
	if len(labels) == 0 {
		labels = []string{"INBOX"}
	}
	m := &Module{
		config:    oauth.Register(config),
		labels:    labels,
		scheduler: timing.NewScheduler(),
	}
	m.RefreshInterval(5 * time.Minute)
	m.Output(func(i Info) bar.Output {
		if i.TotalUnread() == 0 {
			return nil
		}
		return outputs.Textf("Gmail: %d", i.TotalUnread())
	})
	return m
}

// for tests, to wrap the client in a transport that redirects requests.
var wrapForTest func(*http.Client)

func (m *Module) Stream(sink bar.Sink) {
	client, _ := m.config.Client()
	if wrapForTest != nil {
		wrapForTest(client)
	}
	srv, err := gmail.New(client)
	if sink.Error(err) {
		return
	}
	r, err := srv.Users.Labels.List("me").Do()
	if sink.Error(err) {
		return
	}
	labelIDs := map[string]string{}
	for _, l := range r.Labels {
		labelIDs[l.Name] = l.Id
	}
	i, err := m.fetch(srv, labelIDs)
	outf := m.outputFunc.Get().(func(Info) bar.Output)
	nextOutputFunc := m.outputFunc.Next()
	for {
		if sink.Error(err) {
			return
		}
		sink.Output(outf(i))
		select {
		case <-nextOutputFunc:
			nextOutputFunc = m.outputFunc.Next()
			outf = m.outputFunc.Get().(func(Info) bar.Output)
		case <-m.scheduler.Tick():
			i, err = m.fetch(srv, labelIDs)
		}
	}
}

func (m *Module) fetch(srv *gmail.Service, labelIDs map[string]string) (Info, error) {
	i := Info{
		Threads: map[string]int64{},
		Unread:  map[string]int64{},
	}
	for _, l := range m.labels {
		r, err := srv.Users.Labels.Get("me", labelIDs[l]).Do()
		if err != nil {
			return i, err
		}
		i.Threads[l] = r.ThreadsTotal
		i.Unread[l] = r.ThreadsUnread
	}
	return i, nil
}

func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}
