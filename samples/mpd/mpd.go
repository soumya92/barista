package mpd

import (
	"barista.run/bar"
	"barista.run/base/value"
	"barista.run/outputs"
	"github.com/fhs/gompd/mpd"
	"strconv"
)

// Global MPD connection
var conn *mpd.Client

// MPD represents the current state and the current song
type MPD struct {
	status      mpd.Attrs
	currentSong mpd.Attrs
}

// Status returns the current state of MPD as map[string]string aka mpd.Attrs
func (m *MPD) Status() mpd.Attrs {
	return m.status
}

// CurrentSong returns the current song of MPD as map[string]string aka mpd.Attrs
func (m *MPD) CurrentSong() mpd.Attrs {
	return m.currentSong
}

func (m *MPD) setupConnection(host string, port uint64)  {
	parsedPort := strconv.FormatUint(port, 10)
	var err error
	conn, err = mpd.Dial("tcp", host+":"+parsedPort)
	if err != nil {
		conn = nil
	}
	status, _ := conn.Status()
	m.status = status
	if m.status["state"] == "play" {
		currentSong, _ := conn.CurrentSong()
		m.currentSong = currentSong
	}
}

type Module struct {
	host        string
	port        uint64
	outputFunc value.Value
}

// Output configures a module to display the output of a user-defined function.
func (m *Module) Output(outputFunc func(MPD) bar.Output) *Module {
	m.outputFunc.Set(outputFunc)
	return m
}

func (m *Module) Stream(s bar.Sink) {
	var c MPD
	c.setupConnection(m.host, m.port)
}

func New(host string, port uint64) *Module {
	m := &Module {
		host: host,
		port: port,
	}
	m.Output(func(c MPD) bar.Output {
		return outputs.Textf("status: %s current song: %s -- %s", c.status["state"], c.currentSong["Artist"], c.currentSong["Title"])
	})
	return m
}
