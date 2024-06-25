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
	"fmt"
	"os"

	"github.com/soumya92/barista/base/value"
	"github.com/soumya92/barista/modules/volume"

	"github.com/jfreymuth/pulse/proto"
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

func openPulseAudio() (*proto.Client, error) {
	client, _, err := proto.Connect("")
	return client, err
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
	return Sink("@DEFAULT_SINK@")
}

// Source creates a PulseAudio volume module for a named source.
func Source(sourceName string) volume.Provider {
	return Device(sourceName, SourceDevice)
}

// DefaultSource creates a PulseAudio volume module that follows the default source.
func DefaultSource() volume.Provider {
	return Source("@DEFAULT_SOURCE@")
}

type sinkController struct {
	client     *proto.Client
	deviceName string
}

type sourceController struct {
	client     *proto.Client
	deviceName string
}

func (c *sinkController) SetVolume(newVol int64) (err error) {
	return c.client.Request(&proto.SetSinkVolume{
		SinkIndex:      proto.Undefined,
		SinkName:       c.deviceName,
		ChannelVolumes: proto.ChannelVolumes{uint32(newVol)},
	}, nil)
}

func (c *sourceController) SetVolume(newVol int64) (err error) {
	return c.client.Request(&proto.SetSourceVolume{
		SourceIndex:    proto.Undefined,
		SourceName:     c.deviceName,
		ChannelVolumes: proto.ChannelVolumes{uint32(newVol)},
	}, nil)
}

func (c *sinkController) SetMuted(muted bool) (err error) {
	return c.client.Request(&proto.SetSinkMute{
		SinkIndex: proto.Undefined,
		SinkName:  c.deviceName,
		Mute:      muted,
	}, nil)
}

func (c *sourceController) SetMuted(muted bool) (err error) {
	return c.client.Request(&proto.SetSourceMute{
		SourceIndex: proto.Undefined,
		SourceName:  c.deviceName,
		Mute:        muted,
	}, nil)
}

func getVolume(client *proto.Client, deviceName string, deviceType deviceType) (vol volume.Volume, err error) {
	switch deviceType {
	case SinkDevice:
		return getVolumeSink(client, deviceName)
	case SourceDevice:
		return getVolumeSource(client, deviceName)
	default:
		panic(fmt.Sprintf("unexpected device type %v", int(deviceType)))
	}
}

func getVolumeSink(client *proto.Client, deviceName string) (vol volume.Volume, err error) {
	repl := proto.GetSinkInfoReply{}
	err = client.Request(&proto.GetSinkInfo{SinkIndex: proto.Undefined, SinkName: deviceName}, &repl)
	if err != nil {
		return
	}
	return makeVolume(repl.ChannelVolumes, repl.Mute, &sinkController{client, deviceName}), nil
}

func getVolumeSource(client *proto.Client, deviceName string) (vol volume.Volume, err error) {
	repl := proto.GetSourceInfoReply{}
	err = client.Request(&proto.GetSourceInfo{SourceIndex: proto.Undefined, SourceName: deviceName}, &repl)
	if err != nil {
		return
	}
	return makeVolume(repl.ChannelVolumes, repl.Mute, &sourceController{client, deviceName}), nil
}

func makeVolume(channelVolumes proto.ChannelVolumes, mute bool, controller volume.Controller) volume.Volume {
	// Take the volume as the average across all channels.
	var totalVol int64
	for _, ch := range channelVolumes {
		totalVol += int64(ch)
	}
	currentVol := totalVol / int64(len(channelVolumes))
	return volume.MakeVolume(0, int64(proto.VolumeNorm), currentVol, mute, controller)
}

func (m *paModule) Worker(s *value.ErrorValue) {
	client, conn, err := proto.Connect("")
	if s.Error(err) {
		return
	}
	defer conn.Close()

	ch := make(chan struct{}, 1)

	client.Callback = func(val interface{}) {
		switch val.(type) {
		case *proto.SubscribeEvent:
			// When PulseAudio server notifies us about sink/source change,
			// refresh the volume.
			//
			// It's okay if we lose notification something due to channel
			// being full though: this means that refresh is already pending.
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}

	props := proto.PropList{
		"application.name":           proto.PropListString("barista"),
		"application.process.binary": proto.PropListString(os.Args[0]),
		"application.process.id":     proto.PropListString(fmt.Sprintf("%d", os.Getpid())),
	}

	err = client.Request(&proto.SetClientName{Props: props}, nil)
	if s.Error(err) {
		return
	}

	var mask proto.SubscriptionMask
	switch m.deviceType {
	case SinkDevice:
		mask |= proto.SubscriptionMaskSink
	case SourceDevice:
		mask |= proto.SubscriptionMaskSource
	}
	err = client.Request(&proto.Subscribe{Mask: mask}, nil)
	if s.Error(err) {
		return
	}

	for {
		vol, err := getVolume(client, m.deviceName, m.deviceType)
		// Ignore ErrNoSuchEntity because devices may easily go away.
		if err != proto.ErrNoSuchEntity {
			if s.SetOrError(vol, err) {
				return
			}
		}
		<-ch
	}
}
