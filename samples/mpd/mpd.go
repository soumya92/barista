package mpd

import (
	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/outputs"
	"github.com/fhs/gompd/mpd"
	"strconv"
)

type PlaybackStatus string

const (
	// https://www.musicpd.org/doc/html/protocol.html
	Disconnected = PlaybackStatus("")
	Playing      = PlaybackStatus("play")
	Paused       = PlaybackStatus("pause")
	Stopped      = PlaybackStatus("stopped")
)

type Info struct {
	PlaybackStatus PlaybackStatus
	Title          string
	Artist         string
}

// Global MPD connection
var conn *mpd.Client

func (i *Info) setupConnection(host string, port uint64) {
	parsedPort := strconv.FormatUint(port, 10)
	var err error
	conn, err = mpd.Dial("tcp", host+":"+parsedPort)
	if err != nil {
		conn = nil
	}
	status, _ := conn.Status()
	i.PlaybackStatus = PlaybackStatus(status["state"])
	if i.PlaybackStatus == "play" || i.PlaybackStatus == "pause" {
		currentSong, _ := conn.CurrentSong()
		i.Artist = currentSong["Artist"]
		i.Title = currentSong["Title"]
	}
}

type Module struct {
	host       string
	port       uint64
	outputFunc value.Value
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(Info) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) Stream(s bar.Sink) {
	var i Info
	i.setupConnection(m.host, m.port)
	tmp := outputs.Textf("status: %s current song: %s -- %s", i.PlaybackStatus, i.Artist, i.Title)
	s.Output(tmp)
}

func New(host string, port uint64) *Module {
	m := &Module{
		host: host,
		port: port,
	}
	m.Output(func(i Info) bar.Output {
		return outputs.Textf("status: %s current song: %s -- %s", i.PlaybackStatus, i.Artist, i.Title)
	})
	return m
}
