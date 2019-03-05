package mpd

import (
	"github.com/fhs/gompd/mpd"
	"strconv"
)

// MPD represents the current state and the current song
type MPD struct {
	status      mpd.Attrs
	currentSong mpd.Attrs
	host        string
	port        int64
}

// Status returns the current state of MPD as map[string]string aka mpd.Attrs
func (m *MPD) Status() mpd.Attrs {
	return m.status
}

// CurrentSong returns the current song of MPD as map[string]string aka mpd.Attrs
func (m *MPD) CurrentSong() mpd.Attrs {
	return m.currentSong
}

// setHost sets the host for the MPD Dial
func (m *MPD) setHost(host string) {
	m.host = host
}

// setPort sets the port for the MPD Dial
func (m *MPD) setPort(port int64) {
	m.port = port
}

func (m *MPD) setStatus() {
	parsedPort := strconv.FormatInt(m.port, 10)
	conn, _ := mpd.Dial("tcp", m.host+":"+parsedPort)
	status, _ := conn.Status()
	m.status = status
}

func (m *MPD) setCurrentSong() {
	parsedPort := strconv.FormatInt(m.port, 10)
	conn, _ := mpd.Dial("tcp", m.host+":"+parsedPort)
	if m.status["state"] == "play" {
		currentSong, _ := conn.CurrentSong()
		m.currentSong = currentSong
	}
}
