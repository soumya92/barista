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

package systemd

import (
	"testing"
	"time"

	"barista.run/bar"
	"barista.run/base/watchers/dbus"
	"barista.run/outputs"
	testBar "barista.run/testing/bar"
	"barista.run/timing"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	testBar.New(t)
	bus := dbus.SetupTestBus()
	sysd := bus.RegisterService("org.freedesktop.systemd1")

	unit0 := sysd.Object("/org/freedesktop/systemd1/unit/foo_2eservice",
		"org.freedesktop.systemd1.Unit")
	unit0.SetProperties(map[string]interface{}{
		"Id":                   "foo.service",
		"Description":          "A service that foos",
		"ActiveState":          "active",
		"SubState":             "running",
		"StateChangeTimestamp": uint64(timing.Now().UnixNano() / 1000),
	}, dbus.SignalTypeNone)
	srv0 := sysd.Object("/org/freedesktop/systemd1/unit/foo_2eservice",
		"org.freedesktop.systemd1.Service")
	srv0.SetProperties(map[string]interface{}{
		"Type":        "oneshot",
		"MainPID":     uint32(941),
		"ExecMainPID": uint32(931),
	}, dbus.SignalTypeNone)

	unit1 := sysd.Object("/org/freedesktop/systemd1/unit/baz_2dsrv_2eservice",
		"org.freedesktop.systemd1.Unit")
	unit1.SetProperties(map[string]interface{}{
		"Id":          "baz-srv.service",
		"Description": "A service that services baz",
		"ActiveState": "inactive",
		"SubState":    "dead",
	}, dbus.SignalTypeNone)
	srv1 := sysd.Object("/org/freedesktop/systemd1/unit/baz_2dsrv_2eservice",
		"org.freedesktop.systemd1.Service")
	srv1.SetProperties(map[string]interface{}{
		"Type": "oneshot",
	}, dbus.SignalTypeNone)

	m0 := service("foo", dbus.Test)
	m1 := service("baz-srv", dbus.Test)
	testBar.Run(m0, m1)

	testBar.LatestOutput().AssertText([]string{
		"active (running) since 20:47", "inactive (dead)"})

	timing.AdvanceBy(48 * time.Hour)
	unit0.SetProperties(map[string]interface{}{
		"ActiveState": "active",
	}, dbus.SignalTypeChanged)

	testBar.NextOutput().AssertText([]string{
		"active (running) since Nov 25", "inactive (dead)"})

	actionChan := make(chan string, 1)
	unit0.OnElse(func(method string, args ...interface{}) ([]interface{}, error) {
		actionChan <- method
		return nil, nil
	})

	m0.Output(func(i ServiceInfo) bar.Output {
		if i.State == StateActive {
			return outputs.Textf("%s: %d", i.SubState, i.MainPID).
				OnClick(func(e bar.Event) {
					// Don't do this in a real output :)
					switch e.Button {
					case bar.ButtonLeft:
						i.Start()
					case bar.ButtonRight:
						i.Stop()
					case bar.ScrollUp:
						i.Restart()
					case bar.ScrollDown:
						i.Reload()
					}
				})
		}
		return outputs.Textf("%s (%s)", i.State, i.SubState)
	})
	out := testBar.NextOutput()
	out.AssertText([]string{"running: 941", "inactive (dead)"})

	out.At(0).Click(bar.Event{Button: bar.ButtonLeft})
	require.Equal(t, "org.freedesktop.systemd1.Unit.Start", <-actionChan)

	out.At(0).Click(bar.Event{Button: bar.ButtonRight})
	require.Equal(t, "org.freedesktop.systemd1.Unit.Stop", <-actionChan)

	out.At(0).Click(bar.Event{Button: bar.ScrollUp})
	require.Equal(t, "org.freedesktop.systemd1.Unit.Restart", <-actionChan)

	out.At(0).Click(bar.Event{Button: bar.ScrollDown})
	require.Equal(t, "org.freedesktop.systemd1.Unit.Reload", <-actionChan)
}

func TestTimer(t *testing.T) {
	testBar.New(t)
	bus := dbus.SetupTestBus()
	sysd := bus.RegisterService("org.freedesktop.systemd1")

	unit0 := sysd.Object("/org/freedesktop/systemd1/unit/foo_2etimer",
		"org.freedesktop.systemd1.Unit")
	unit0.SetProperties(map[string]interface{}{
		"Id":                   "foo.timer",
		"Description":          "A timer for the foo service",
		"ActiveState":          "active",
		"SubState":             "waiting",
		"StateChangeTimestamp": uint64(timing.Now().Add(-720*time.Hour).UnixNano() / 1000),
	}, dbus.SignalTypeNone)
	tim0 := sysd.Object("/org/freedesktop/systemd1/unit/foo_2etimer",
		"org.freedesktop.systemd1.Timer")
	tim0.SetProperties(map[string]interface{}{
		"Unit":                   "foo.service",
		"NextElapseUSecRealtime": uint64(timing.Now().Add(6*time.Hour).UnixNano() / 1000),
	}, dbus.SignalTypeNone)

	m0 := timer("foo", dbus.Test)
	testBar.Run(m0)

	testBar.LatestOutput().AssertText([]string{
		"foo.service@Nov 26, 02:47 (last:never)"})

	timing.AdvanceBy(7 * time.Hour)
	tim0.SetProperties(map[string]interface{}{
		"LastTriggerUSec":        uint64(timing.Now().Add(-1*time.Hour).UnixNano() / 1000),
		"NextElapseUSecRealtime": uint64(timing.Now().Add(23*time.Hour).UnixNano() / 1000),
	}, dbus.SignalTypeNone)
	unit0.SetPropertyForTest("ActiveState", "active", dbus.SignalTypeChanged)

	testBar.LatestOutput().AssertText([]string{
		"foo.service@Nov 27, 02:47 (last:Nov 26, 02:47)"})

	m0.Output(func(i TimerInfo) bar.Output {
		return outputs.Textf("%s@%s", i.Unit, i.NextTrigger.Format("15:04"))
	})
	testBar.LatestOutput().AssertText([]string{"foo.service@02:47"})
}
