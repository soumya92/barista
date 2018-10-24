// Copyright 2017 Google Inc.
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

/*
Package shell provides modules to display the output of shell commands.
It supports both long-running commands, where the output is the last line,
e.g. dmesg or tail -f /var/log/some.log, and repeatedly running commands,
e.g. whoami, date +%s.
*/
package shell // import "barista.run/modules/shell"

import (
	"os/exec"
	"strings"
	"time"

	"barista.run/bar"
	"barista.run/base/notifier"
	"barista.run/base/value"
	"barista.run/outputs"
	"barista.run/timing"
)

// Module represents a shell module that updates on a timer or on demand.
type Module struct {
	cmd       string
	args      []string
	outf      value.Value // of func(string) bar.Output
	notifyCh  <-chan struct{}
	notifyFn  func()
	scheduler *timing.Scheduler
}

// New constructs a new shell module.
func New(cmd string, args ...string) *Module {
	m := &Module{cmd: cmd, args: args}
	m.notifyFn, m.notifyCh = notifier.New()
	m.scheduler = timing.NewScheduler()
	m.outf.Set(func(text string) bar.Output {
		return outputs.Text(text)
	})
	return m
}

// Stream starts the module.
func (m *Module) Stream(s bar.Sink) {
	out, err := exec.Command(m.cmd, m.args...).Output()
	outf := m.outf.Get().(func(string) bar.Output)
	for {
		if s.Error(err) {
			return
		}
		s.Output(outf(strings.TrimSpace(string(out))))
		select {
		case <-m.outf.Next():
			outf = m.outf.Get().(func(string) bar.Output)
		case <-m.notifyCh:
			out, err = exec.Command(m.cmd, m.args...).Output()
		case <-m.scheduler.C:
			out, err = exec.Command(m.cmd, m.args...).Output()
		}
	}
}

// Output sets the output format. The format func will be passed the entire
// trimmed output from the command once it's done executing. To process output
// by lines, see Tail().
func (m *Module) Output(format func(string) bar.Output) *Module {
	m.outf.Set(format)
	return m
}

// Every sets the refresh interval for the module. The command will be executed
// repeatedly at the given interval, and the output updated. A zero interval
// stops automatic repeats (but Refresh will still work).
func (m *Module) Every(interval time.Duration) *Module {
	if interval == 0 {
		m.scheduler.Stop()
	} else {
		m.scheduler.Every(interval)
	}
	return m
}

// Refresh executes the command and updates the output.
func (m *Module) Refresh() {
	m.notifyFn()
}
