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

package pulseaudio

import (
	"C"
	"fmt"
	"os"

	"barista.run/base/value"
	"barista.run/modules/volume"

	"github.com/godbus/dbus/v5"
)

type deviceType int

const (
	// SinkDevice represents devices used for audio output, e.g. headphones.
	SinkDevice deviceType = iota
	// SourceDevice represents devices used for audio input, e.g. microphones.
	SourceDevice
)

func (deviceType deviceType) String() string {
	return ([]string{"Sink", "Source"})[deviceType]
}

// PulseAudio implementation.
type paModule struct {
	deviceName string
	deviceType deviceType
}

type paController struct {
	device dbus.BusObject
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

// Device creates a PulseAduio volume module for a named device that can either be a sink or a source.
func Device(deviceName string, deviceType deviceType) volume.Provider {
	return &paModule{deviceName: deviceName, deviceType: deviceType}
}

// Sink creates a PulseAudio volume module for a named sink.
func Sink(sinkName string) volume.Provider {
	return Device(sinkName, SinkDevice)
}

// DefaultSink creates a PulseAudio volume module that follows the default sink.
func DefaultSink() volume.Provider {
	return Sink("")
}

// Source creates a PulseAudio volume module for a named source.
func Source(sourceName string) volume.Provider {
	return Device(sourceName, SourceDevice)
}

// DefaultSource creates a PulseAudio volume module that follows the default source.
func DefaultSource() volume.Provider {
	return Source("")
}

func (c *paController) SetVolume(newVol int64) error {
	return c.device.Call(
		"org.freedesktop.DBus.Properties.Set",
		0,
		"org.PulseAudio.Core1.Device",
		"Volume",
		dbus.MakeVariant([]uint32{uint32(newVol)}),
	).Err
}

func (c *paController) SetMuted(muted bool) error {
	return c.device.Call(
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

func openDevice(conn *dbus.Conn, core dbus.BusObject, devicePath dbus.ObjectPath, deviceType deviceType) (dbus.BusObject, error) {
	device := conn.Object("org.PulseAudio.Core1."+deviceType.String(), devicePath)
	if err := listen(core, "Device.VolumeUpdated", devicePath); err != nil {
		return nil, err
	}
	return device, listen(core, "Device.MuteUpdated", devicePath)
}

func openDeviceByName(conn *dbus.Conn, core dbus.BusObject, name string, deviceType deviceType) (dbus.BusObject, error) {
	var path dbus.ObjectPath
	err := core.Call("org.PulseAudio.Core1.Get"+deviceType.String()+"ByName", 0, name).Store(&path)
	if err != nil {
		return nil, err
	}
	return openDevice(conn, core, path, deviceType)
}

func openFallbackDevice(conn *dbus.Conn, core dbus.BusObject, deviceType deviceType) (dbus.BusObject, error) {
	path, err := core.GetProperty("org.PulseAudio.Core1.Fallback" + deviceType.String())
	if err != nil {
		return nil, err
	}
	return openDevice(conn, core, path.Value().(dbus.ObjectPath), deviceType)
}

func getVolume(device dbus.BusObject) (volume.Volume, error) {
	max, err := device.GetProperty("org.PulseAudio.Core1.Device.BaseVolume")
	if err != nil {
		return volume.Volume{}, err
	}
	maxVol := int64(max.Value().(uint32))

	vol, err := device.GetProperty("org.PulseAudio.Core1.Device.Volume")
	if err != nil {
		return volume.Volume{}, err
	}

	// Take the volume as the average across all channels.
	var totalVol int64
	channels := vol.Value().([]uint32)
	for _, ch := range channels {
		totalVol += int64(ch)
	}
	currentVol := totalVol / int64(len(channels))

	mute, err := device.GetProperty("org.PulseAudio.Core1.Device.Mute")
	if err != nil {
		return volume.Volume{}, err
	}
	muted := mute.Value().(bool)

	return volume.MakeVolume(0, maxVol, currentVol, muted, &paController{device}), nil
}

func (m *paModule) Worker(s *value.ErrorValue) {
	conn, err := openPulseAudio()
	if s.Error(err) {
		return
	}
	defer conn.Close()

	core := conn.Object("org.PulseAudio.Core1", "/org/pulseaudio/core1")

	var device dbus.BusObject
	if m.deviceName != "" {
		device, err = openDeviceByName(conn, core, m.deviceName, m.deviceType)
	} else {
		device, err = openFallbackDevice(conn, core, m.deviceType)
		if err == nil {
			err = listen(core, "Fallback"+m.deviceType.String()+"Updated")
		}
	}
	if s.Error(err) {
		return
	}
	if s.SetOrError(getVolume(device)) {
		return
	}

	signals := make(chan *dbus.Signal, 10)
	conn.Signal(signals)

	// Listen for signals from D-Bus, and update appropriately.
	for signal := range signals {
		// If the fallback device changed, open the new one.
		if m.deviceName == "" && signal.Path == core.Path() {
			device, err = openFallbackDevice(conn, core, m.deviceType)
			if s.Error(err) {
				return
			}
		}
		if s.SetOrError(getVolume(device)) {
			return
		}
	}
}
