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
	"os"
	"path"
	"strings"
	"sync"

	"github.com/soumya92/barista/bar"
	"github.com/soumya92/barista/base/value"
	"github.com/soumya92/barista/outputs"

	"github.com/maximbaz/yubikey-touch-detector/detector"
	ykNotifier "github.com/maximbaz/yubikey-touch-detector/notifier"
)

// Module represents a yubikey barista module that shows an indicator whenever
// the plugged-in yubikey is waiting for user input.
type Module struct {
	gpgPubringPath string
	outputFunc     value.Value // of func(bool, bool) bar.Output
}

// ForPath constructs a yubikey module with the given path to the gpg keyring.
func ForPath(gpgPubringPath string) *Module {
	m := &Module{
		gpgPubringPath: gpgPubringPath,
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

// New constructs a new yubikey module using the default paths for the u2f
// pending file and gpg keyring.
func New() *Module {
	gpgHome := os.Getenv("GNUPGHOME")
	if gpgHome != "" {
		return ForPath(path.Join(gpgHome, "pubring.kbx"))
	}
	return ForPath(os.ExpandEnv("$HOME/.gnupg/pubring.kbx"))
}

// Output sets the output format for the module.
func (m *Module) Output(outputFunc func(bool, bool) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

// Stream starts the module.
func (m *Module) Stream(sink bar.Sink) {
	ykChan := make(chan ykNotifier.Message, 10)
	notifiers := new(sync.Map)
	notifiers.Store("barista", ykChan)

	requestGPGCheck := make(chan bool)
	go detector.CheckGPGOnRequest(requestGPGCheck, notifiers)
	go detector.WatchU2F(notifiers)
	go detector.WatchGPG(m.gpgPubringPath, requestGPGCheck)

	exits := new(sync.Map)
	go detector.WatchSSH(requestGPGCheck, exits)
	defer func(exits *sync.Map) {
		exits.Range(func(_, value any) bool {
			if ch, ok := value.(chan bool); ok {
				ch <- true
			}
			return true
		})
	}(exits)

	gpg := false
	u2f := false
	outf := m.outputFunc.Get().(func(bool, bool) bar.Output)
	nextOutputFunc, done := m.outputFunc.Subscribe()
	defer done()
	for {
		sink.Output(outf(gpg, u2f))
		select {
		case msg := <-ykChan:
			switch msg {
			case ykNotifier.GPG_ON:
				gpg = true
			case ykNotifier.GPG_OFF:
				gpg = false
			case ykNotifier.U2F_ON:
				u2f = true
			case ykNotifier.U2F_OFF:
				u2f = false
			}
		case <-nextOutputFunc:
			outf = m.outputFunc.Get().(func(bool, bool) bar.Output)
		}
	}
}
