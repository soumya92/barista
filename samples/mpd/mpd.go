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

type MPD struct {
	watcher            *mpd.Watcher
	Message            chan *Info
}

type Info struct {
	PlaybackStatus     PlaybackStatus
	Title              string
	Artist             string
	PlaybackStatusIcon PlaybackStatusIcon
}

// Paused returns true if the media player is connected and paused.
func (i Info) Paused() bool { return i.PlaybackStatus == Paused }

// Playing returns true if the media player is connected and playing media.
func (i Info) Playing() bool { return i.PlaybackStatus == Playing }

// Stopped returns true if the media player is connected but stopped.
func (i Info) Stopped() bool { return i.PlaybackStatus == Stopped }

// Connected returns true if the media player is connected.
func (i Info) Connected() bool { return i.PlaybackStatus != Disconnected }


func connect(host string) *mpd.Client {
	if host == "" {
		host = "127.0.0.1:6600"
	}
	conn, err := mpd.Dial("tcp", host)
	if err != nil {
		conn = nil
	}
	return conn
}

func getStatus(host string) (i *Info){
	conn := connect(host)
	status, _ := conn.Status()
	i.PlaybackStatus = PlaybackStatus(status["state"])
	if i.Playing() {
		i.PlaybackStatusIcon = PlayingIcon
	} else if i.Paused() {
		i.PlaybackStatusIcon = PausedIcon
	}
	if i.Playing() || i.Paused() {
		currentSong, _ := conn.CurrentSong()
		i.Artist = currentSong["Artist"]
		i.Title = currentSong["Title"]
	}
	return i
}

func (m *MPD) watch(host string) {
	watcher, err := mpd.NewWatcher("tcp", host, "")
	if err != nil {
		m.watcher = nil
		return
	} else {
		m.watcher = watcher
	}
	for {
		m.Message <- getStatus(host)
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
	mpdDaemon := &MPD{}
	go func() {
		mpdDaemon.watch(m.host)
	}()
	i := <-mpdDaemon.Message
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
