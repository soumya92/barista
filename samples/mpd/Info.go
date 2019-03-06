package mpd

// Paused returns true if the media player is connected and paused.
func (i Info) Paused() bool { return i.PlaybackStatus == Paused }

// Playing returns true if the media player is connected and playing media.
func (i Info) Playing() bool { return i.PlaybackStatus == Playing }

// Stopped returns true if the media player is connected but stopped.
func (i Info) Stopped() bool { return i.PlaybackStatus == Stopped }

// Connected returns true if the media player is connected.
func (i Info) Connected() bool { return i.PlaybackStatus != Disconnected }

func (i *Info) updateInfo() {
	status, _ := i.conn.Status()
	i.PlaybackStatus = PlaybackStatus(status["state"])
	if i.Playing() {
		i.PlaybackStatusIcon = PlayingIcon
	} else if i.Paused() {
		i.PlaybackStatusIcon = PausedIcon
	}
	if i.Playing() || i.Paused() {
		currentSong, _ := i.conn.CurrentSong()
		i.Artist = currentSong["Artist"]
		i.Title = currentSong["Title"]
	}
}
