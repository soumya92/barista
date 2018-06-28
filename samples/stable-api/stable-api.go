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

// stable-api demonstrates a bar that exercises barista's stable API.
package main

import (
	"os/exec"
	"syscall"
	"time"

	"github.com/dustin/go-humanize"

	"github.com/soumya92/barista"
	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/timing"
)

type simpleClockModule struct {
	format   string
	interval time.Duration
}

func (s simpleClockModule) Stream(sink bar.Sink) {
	for {
		now := timing.Now()
		sink.Output(bar.TextSegment(now.Format(s.format)))
		next := now.Add(s.interval).Truncate(s.interval)
		time.Sleep(next.Sub(now))
	}
}

type diskSpaceModule string

func (d diskSpaceModule) Stream(sink bar.Sink) {
	sch := timing.NewScheduler().Every(5 * time.Second)
	for {
		var stat syscall.Statfs_t
		err := syscall.Statfs(string(d), &stat)
		if sink.Error(err) {
			return
		}
		sink.Output(bar.TextSegment(
			humanize.IBytes(stat.Bavail * uint64(stat.Bsize)),
		))
		<-sch.Tick()
	}
}

func (d diskSpaceModule) Click(e bar.Event) {
	exec.Command("xdg-open", string(d)).Run()
}

func main() {
	panic(barista.Run(
		diskSpaceModule("/home"),
		simpleClockModule{"Mon Jan 02", time.Hour},
		simpleClockModule{"15:04:05", time.Second},
	))
}
