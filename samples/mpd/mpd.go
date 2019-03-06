package mpd

import (
	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/outputs"
	"github.com/fhs/gompd/mpd"
)

type PlaybackStatus string

const (
	// https://www.musicpd.org/doc/html/protocol.html
	Disconnected = PlaybackStatus("")
	Playing      = PlaybackStatus("play")
	Paused       = PlaybackStatus("pause")
	Stopped      = PlaybackStatus("stopped")
)

type PlaybackStatusIcon string

const (
	PlayingIcon = PlaybackStatusIcon("►")
	PausedIcon  = PlaybackStatusIcon("⏸")
)

type Info struct {
	PlaybackStatus     PlaybackStatus
	Title              string
	Artist             string
	PlaybackStatusIcon PlaybackStatusIcon
	conn               *mpd.Client
	watcher            *mpd.Watcher
}

func (i *Info) setupConnection(host string) {
	if host == "" {
		host = "127.0.0.1:6600"
	}
	conn, err := mpd.Dial("tcp", host)
	if err != nil {
		conn = nil
	}
	i.conn = conn
	watcher, err := mpd.NewWatcher("tcp", host, "")
	if err != nil {
		i.watcher = nil
	} else {
		i.watcher = watcher
	}
}

type Module struct {
	host       string
	outputFunc value.Value
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) Stream(s bar.Sink) {
	var i Info
	i.setupConnection(m.host)
	i.updateInfo()
	tmp := outputs.Textf("%s %s - %s", i.PlaybackStatusIcon, i.Artist, i.Title)
	s.Output(tmp)
}

func New(host string) *Module {
	m := &Module{
		host: host,
	}
	m.Output(func(i Info) bar.Output {
		return outputs.Textf("%s %s - %s", i.PlaybackStatusIcon, i.Artist, i.Title)
	})
	return m
}
