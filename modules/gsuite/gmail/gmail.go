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

// Info represents the unread and total thread counts for labels.
// The keys are the names (not IDs) of the labels, and the values are the thread
// counts (Threads is total threads, while Unread is just unread threads).
type Info struct {
	Threads map[string]int64
	Unread  map[string]int64
}

// TotalUnread is the total number of unread threads across all labels. (as set
// during construction).
func (i Info) TotalUnread() int64 {
	t := int64(0)
	for _, u := range i.Unread {
		t += u
	}
	return t
}

// TotalThreads is the total number of threads across all configured labels.
func (i Info) TotalThreads() int64 {
	t := int64(0)
	for _, c := range i.Threads {
		t += c
	}
	return t
}

// Module represents a Gmail barista module.
type Module struct {
	config     *oauth.Config
	labels     []string
	scheduler  *timing.Scheduler
	outputFunc value.Value // of func(Info) bar.Output
}

// New creates a gmail module from the given oauth config, that fetches unread
// and total thread counts for the given set of labels.
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

// Stream starts the module.
func (m *Module) Stream(sink bar.Sink) {
	client, _ := m.config.Client()
	if wrapForTest != nil {
		wrapForTest(client)
	}
	srv, _ := gmail.New(client)
	r, err := srv.Users.Labels.List("me").Do()
	if sink.Error(err) {
		return
	}
	labelIDs := map[string]string{}
	for _, l := range r.Labels {
		labelIDs[l.Name] = l.Id
	}
	i, err := fetch(srv, m.labels, labelIDs)
	outf := m.outputFunc.Get().(func(Info) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()
	for {
		if sink.Error(err) {
			return
		}
		sink.Output(outf(i))
		select {
		case <-nextOutputFunc:
			outf = m.outputFunc.Get().(func(Info) bar.Output)
		case <-m.scheduler.C:
			i, err = fetch(srv, m.labels, labelIDs)
		}
	}
}

func fetch(srv *gmail.Service, labels []string, labelIDs map[string]string) (Info, error) {
	i := Info{
		Threads: map[string]int64{},
		Unread:  map[string]int64{},
	}
	for _, l := range labels {
		r, err := srv.Users.Labels.Get("me", labelIDs[l]).Do()
		if err != nil {
			return i, err
		}
		i.Threads[l] = r.ThreadsTotal
		i.Unread[l] = r.ThreadsUnread
	}
	return i, nil
}

// Output sets the output format for the module.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// RefreshInterval sets the interval between consecutive checks for new mail.
func (m *Module) RefreshInterval(interval time.Duration) *Module {
	m.scheduler.Every(interval)
	return m
}
