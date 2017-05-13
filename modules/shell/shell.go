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
package shell

import (
	"bufio"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/modules/base"
	"github.com/soumya92/barista/outputs"
)

type tailModule struct {
	*base.Base
	cmd  string
	args []string
}

// Tail constructs a module that displays the last line of output from
// a long running command. Use the reformat module to adjust the output
// if necessary.
func Tail(cmd string, args ...string) base.WithClickHandler {
	return &tailModule{
		Base: base.New(),
		cmd:  cmd,
		args: args,
	}
}

func (m *tailModule) Stream() <-chan bar.Output {
	go m.worker()
	return m.Base.Stream()
}

func (m *tailModule) worker() {
	cmd := exec.Command(m.cmd, m.args...)
	// Prevent SIGUSR for bar pause/resume from propagating to the
	// child process. Some commands don't play nice with signals.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
	stdout, err := cmd.StdoutPipe()
	if m.Error(err) {
		return
	}
	if m.Error(cmd.Start()) {
		return
	}
	m.OnUpdate(func() {})
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		m.Output(outputs.Text(scanner.Text()))
	}
	m.Error(cmd.Wait())
	// If the process died, the next update should restart it.
	// Since we clear onUpdate when the process starts successfully,
	// updates while the process is running are no-ops.
	m.OnUpdate(m.worker)
}

// Every constructs a module that runs the given command with the
// specified interval and displays the commands output in the bar.
func Every(interval time.Duration, cmd string, args ...string) base.WithClickHandler {
	m := base.New()
	m.OnUpdate(func() {
		commandOutputToModule(m, cmd, args...)
	})
	m.UpdateEvery(interval)
	return m
}

// Once constructs a static module that displays the output of
// the given command in the bar.
func Once(cmd string, args ...string) base.WithClickHandler {
	m := base.New()
	commandOutputToModule(m, cmd, args...)
	return m
}

// commandOutputToModule runs the command and sends the output or error
// as appropriate to the base module.
func commandOutputToModule(module *base.Base, cmd string, args ...string) {
	out, err := exec.Command(cmd, args...).Output()
	if module.Error(err) {
		return
	}
	strOut := strings.TrimSpace(string(out))
	module.Output(outputs.Text(strOut))
}
