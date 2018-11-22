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

package volume

import (
	"C"
	"fmt"
	"os"

	"barista.run/base/value"
	l "barista.run/logging"

	"github.com/godbus/dbus"
)

// PulseAudio implementation.
type paModule struct {
	sinkName string
}

type paController struct {
	sink dbus.BusObject
}

func dialAndAuth(addr string) (*dbus.Conn, error) {
	conn, err := dbus.Dial(addr)
	if err != nil {
		return nil, err
	}
	err = conn.Auth(nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func openPulseAudio() (*dbus.Conn, error) {
	// Pulse defaults to creating its socket in a well-known place under
	// XDG_RUNTIME_DIR. For Pulse instances created by systemd, this is the
	// only reliable way to contact Pulse via D-Bus, since Pulse is created
	// on a per-user basis, but the session bus is created once for every
	// session, and a user can have multiple sessions.
	xdgDir := os.Getenv("XDG_RUNTIME_DIR")
	if xdgDir != "" {
		addr := fmt.Sprintf("unix:path=%s/pulse/dbus-socket", xdgDir)
		return dialAndAuth(addr)
	}

	// Couldn't find the PulseAudio bus on the fast path, so look for it
	// by querying the session bus.
	bus, err := dbus.SessionBusPrivate()
	if err != nil {
		return nil, err
	}
	defer bus.Close()
	err = bus.Auth(nil)
	if err != nil {
		return nil, err
	}

	locator := bus.Object("org.PulseAudio1", "/org/pulseaudio/server_lookup1")
	path, err := locator.GetProperty("org.PulseAudio.ServerLookup1.Address")
	if err != nil {
		return nil, err
	}

	return dialAndAuth(path.Value().(string))
}

// Sink creates a PulseAudio volume module for a named sink.
func Sink(sinkName string) *Module {
	m := createModule(&paModule{sinkName: sinkName})
	if sinkName == "" {
		sinkName = "default"
	}
	l.Labelf(m, "pulse:%s", sinkName)
	return m
}

// DefaultSink creates a PulseAudio volume module that follows the default sink.
func DefaultSink() *Module {
	return Sink("")
}

func (c *paController) setVolume(_ Volume, newVol int64) error {
	return c.sink.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"org.PulseAudio.Core1.Device",
		"Volume",
		dbus.MakeVariant([]uint32{uint32(newVol)}),
	).Err
}

func (c *paController) setMuted(_ Volume, muted bool) error {
	return c.sink.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"org.PulseAudio.Core1.Device",
		"Mute",
		dbus.MakeVariant(muted),
	).Err
}

func listen(core dbus.BusObject, signal string, objects ...dbus.ObjectPath) error {
	return core.Call(
		"org.PulseAudio.Core1.ListenForSignal",
		0,
		"org.PulseAudio.Core1."+signal,
		objects,
	).Err
}

func openSink(conn *dbus.Conn, core dbus.BusObject, sinkPath dbus.ObjectPath) (dbus.BusObject, error) {
	sink := conn.Object("org.PulseAudio.Core1.Sink", sinkPath)
	if err := listen(core, "Device.VolumeUpdated", sinkPath); err != nil {
		return nil, err
	}
	return sink, listen(core, "Device.MuteUpdated", sinkPath)
}

func openSinkByName(conn *dbus.Conn, core dbus.BusObject, name string) (dbus.BusObject, error) {
	var path dbus.ObjectPath
	err := core.Call("org.PulseAudio.Core1.GetSinkByName", 0, name).Store(&path)
	if err != nil {
		return nil, err
	}
	return openSink(conn, core, path)
}

func openFallbackSink(conn *dbus.Conn, core dbus.BusObject) (dbus.BusObject, error) {
	path, err := core.GetProperty("org.PulseAudio.Core1.FallbackSink")
	if err != nil {
		return nil, err
	}
	return openSink(conn, core, path.Value().(dbus.ObjectPath))
}

func getVolume(sink dbus.BusObject) (Volume, error) {
	v := Volume{}
	v.Min = 0

	max, err := sink.GetProperty("org.PulseAudio.Core1.Device.BaseVolume")
	if err != nil {
		return v, err
	}
	v.Max = int64(max.Value().(uint32))

	vol, err := sink.GetProperty("org.PulseAudio.Core1.Device.Volume")
	if err != nil {
		return v, err
	}

	// Take the volume as the average across all channels.
	var totalVol int64
	channels := vol.Value().([]uint32)
	for _, ch := range channels {
		totalVol += int64(ch)
	}
	v.Vol = totalVol / int64(len(channels))

	mute, err := sink.GetProperty("org.PulseAudio.Core1.Device.Mute")
	if err != nil {
		return v, err
	}
	v.Mute = mute.Value().(bool)
	v.controller = &paController{sink}
	return v, nil
}

func (m *paModule) worker(s *value.ErrorValue) {
	conn, err := openPulseAudio()
	if s.Error(err) {
		return
	}
	defer conn.Close()

	core := conn.Object("org.PulseAudio.Core1", "/org/pulseaudio/core1")

	var sink dbus.BusObject
	if m.sinkName != "" {
		sink, err = openSinkByName(conn, core, m.sinkName)
	} else {
		sink, err = openFallbackSink(conn, core)
		if err != nil {
			err = listen(core, "FallbackSinkUpdated")
		}
	}
	if s.Error(err) {
		return
	}
	if s.SetOrError(getVolume(sink)) {
		return
	}

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	// Listen for signals from D-Bus, and update appropriately.
	for signal := range signals {
		// If the fallback sink changed, open the new one.
		if m.sinkName == "" && signal.Path == core.Path() {
			sink, err = openFallbackSink(conn, core)
			if s.Error(err) {
				return
			}
		}
		if s.SetOrError(getVolume(sink)) {
			return
		}
	}
}
