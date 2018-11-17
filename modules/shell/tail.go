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

package shell

import (
	"bufio"
	"os/exec"
	"syscall"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/outputs"
)

// TailModule represents a bar.Module that displays the last line of output from
// a shell command in the bar.
type TailModule struct {
	cmd  string
	args []string
	outf value.Value // of func(string) bar.Output
}

// Tail constructs a module that displays the last line of output from a long
// running command.
func Tail(cmd string, args ...string) *TailModule {
	t := &TailModule{cmd: cmd, args: args}
	t.outf.Set(func(text string) bar.Output {
		return outputs.Text(text)
	})
	return t
}

// Stream starts the module.
func (m *TailModule) Stream(s bar.Sink) {
	cmd := exec.Command(m.cmd, m.args...)
	// Prevent SIGUSR for bar pause/resume from propagating to the
	// child process. Some commands don't play nice with signals.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}
	stdout, err := cmd.StdoutPipe()
	if s.Error(err) {
		return
	}
	if s.Error(cmd.Start()) {
		return
	}
	var out *string
	outf := m.outf.Get().(func(string) bar.Output)
	errChan := make(chan error)
	outChan := make(chan string)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			outChan <- scanner.Text()
		}
		errChan <- cmd.Wait()
	}()
	for {
		select {
		case e := <-errChan:
			s.Error(e)
			return
		case <-m.outf.Next():
			outf = m.outf.Get().(func(string) bar.Output)
		case txt := <-outChan:
			out = &txt
		}
		if out != nil {
			s.Output(outf(*out))
		}
	}
}

// Output sets the output format for each line of output.
func (m *TailModule) Output(format func(string) bar.Output) *TailModule {
	m.outf.Set(format)
	return m
}
