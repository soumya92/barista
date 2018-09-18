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

// Package yubikey provides an indicator for a waiting yubikey.
package yubikey

import (
	"fmt"
	"os"
	"path"
	"strings"

	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/outputs"

	"github.com/maximbaz/yubikey-touch-detector/detector"
	"github.com/maximbaz/yubikey-touch-detector/notifier"
)

type Module struct {
	u2fAuthPendingPath string
	gpgPubringPath     string
	outputFunc         value.Value // of func(bool, bool) bar.Output
}

var U2FAuthPendingPath = fmt.Sprintf("/var/run/user/%d/pam-u2f-authpending", os.Getuid())

func ForPaths(u2fAuthPendingPath string, gpgPubringPath string) *Module {
	m := &Module{
		u2fAuthPendingPath: u2fAuthPendingPath,
		gpgPubringPath:     gpgPubringPath,
	}
	m.Output(func(gpg, u2f bool) bar.Output {
		reason := []string{}
		if gpg {
			reason = append(reason, "GPG")
		}
		if u2f {
			reason = append(reason, "U2F")
		}
		if len(reason) == 0 {
			return nil
		}
		return outputs.Textf("[YK: %s]", strings.Join(reason, ","))
	})
	return m
}

func New() *Module {
	gpgHome := os.Getenv("GNUPGHOME")
	if gpgHome != "" {
		return ForPaths(U2FAuthPendingPath, path.Join(gpgHome, "pubring.kbx"))
	}
	return ForPaths(U2FAuthPendingPath, os.ExpandEnv("$HOME/.gnupg/pubring.kbx"))
}

func (m *Module) Output(outputFunc func(bool, bool) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) Stream(sink bar.Sink) {
	ykChan := make(chan notifier.Message, 10)
	notifiers := map[string]chan notifier.Message{"barista": ykChan}

	requestGPGCheck := make(chan bool)
	go detector.CheckGPGOnRequest(requestGPGCheck, notifiers)
	go detector.WatchU2F(m.u2fAuthPendingPath, notifiers)
	go detector.WatchGPG(m.gpgPubringPath, requestGPGCheck)

	exits := make(map[string]chan bool)
	go detector.WatchSSH(requestGPGCheck, exits)
	defer func(exits map[string]chan bool) {
		for _, c := range exits {
			c <- true
		}
	}(exits)

	gpg := false
	u2f := false
	outf := m.outputFunc.Get().(func(bool, bool) bar.Output)
	nextOutputFunc := m.outputFunc.Next()
	for {
		sink.Output(outf(gpg, u2f))
		select {
		case msg := <-ykChan:
			switch msg {
			case notifier.GPG_ON:
				gpg = true
			case notifier.GPG_OFF:
				gpg = false
			case notifier.U2F_ON:
				u2f = true
			case notifier.U2F_OFF:
				u2f = false
			}
		case <-nextOutputFunc:
			nextOutputFunc = m.outputFunc.Next()
			outf = m.outputFunc.Get().(func(bool, bool) bar.Output)
		}
	}
}
