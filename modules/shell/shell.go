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
	"github.com/soumya92/barista/base"
	"github.com/soumya92/barista/modules/funcs"
	"github.com/soumya92/barista/outputs"
)

// TailModule represents a bar.Module that displays the last line
// of output from a shell command in the bar.
type TailModule struct {
	base.SimpleClickHandler
	cmd  string
	args []string
}

// Tail constructs a module that displays the last line of output from
// a long running command. Use the reformat module to adjust the output
// if necessary.
func Tail(cmd string, args ...string) *TailModule {
	return &TailModule{cmd: cmd, args: args}
}

// Stream starts the module.
func (m *TailModule) Stream() <-chan bar.Output {
	ch := base.NewChannel()
	go m.worker(ch)
	return ch
}

func (m *TailModule) worker(ch base.Channel) {
	cmd := exec.Command(m.cmd, m.args...)
	// Prevent SIGUSR for bar pause/resume from propagating to the
	// child process. Some commands don't play nice with signals.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
	stdout, err := cmd.StdoutPipe()
	if ch.Error(err) {
		return
	}
	if ch.Error(cmd.Start()) {
		return
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		txt := scanner.Text()
		ch.Output(outputs.Text(txt))
	}
	if ch.Error(cmd.Wait()) {
		return
	}
	close(ch)
}

// Every constructs a module that runs the given command at the
// specified interval and displays the commands output in the bar.
func Every(interval time.Duration, cmd string, args ...string) *funcs.RepeatingModule {
	return funcs.Every(interval, func(ch funcs.Channel) {
		commandOutput(ch, cmd, args...)
	})
}

// Once constructs a static module that displays the output of
// the given command in the bar.
func Once(cmd string, args ...string) *funcs.OnceModule {
	return funcs.Once(func(ch funcs.Channel) {
		commandOutput(ch, cmd, args...)
	})
}

// OnClick constructs a module that displays the output of the given
// command in the bar, and refreshes the output on click.
func OnClick(cmd string, args ...string) bar.Module {
	return funcs.OnClick(func(ch funcs.Channel) {
		commandOutput(ch, cmd, args...)
	})
}

// commandOutput runs the command and sends the output or error to the channel.
func commandOutput(ch funcs.Channel, cmd string, args ...string) {
	out, err := exec.Command(cmd, args...).Output()
	if ch.Error(err) {
		return
	}
	strOut := strings.TrimSpace(string(out))
	ch.Output(outputs.Text(strOut))
}
