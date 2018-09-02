---
title: Media Player
---

Show information about the currently playing track: `media.New("rhythmbox")`.

The name of the player should match what it uses to register its MPRIS interface with D-Bus. Use
[`playerctl --list-all`](https://github.com/acrisci/playerctl) with the media player running to get
the name to use here.

Not all media players register with D-Bus by default. Some require a setting to be enabled, while
others may need a plugin to be installed. Searching for MPRIS or D-Bus in your player's docs should
yield instructions on how to enable D-Bus registration.

## Configuration

* `Output(func(Info) bar.Output)`: Sets the output format.

  If a segment does not have a click handler, the module will set a default click handler, which:
  - Toggles play/pause on left click
  - Switches to the previous/next track on back/forward mouse buttons
  - Seeks backward/forward on scroll up/down

## Example

<div class="module-example-out"><span>|&lt;</span><span>||</span><span>&gt;|</span><span>30s/4m33s</span></div>
Demonstrating click handlers and multi-segment output:

```go
func ifLeft(dofn func()) func(bar.Event) {
	return func(e bar.Event) {
		if e.Button == bar.ButtonLeft {
			dofn()
		}
	}
}

media.New("rhythmbox").Output(func(m media.Info) bar.Output {
	if !m.Connected() {
		return nil
	}
	out := new(outputs.SegmentGroup)
	out.Append(outputs.Text("|<").OnClick(ifLeft(m.Previous)))
	if m.Playing() {
		out.Append(outputs.Text("||").OnClick(ifLeft(m.Pause)))
	} else {
		out.Append(outputs.Text(">").OnClick(ifLeft(m.Play)))
	}
	out.Append(outputs.Text(">|").OnClick(ifLeft(m.Next)))
	if m.Playing() {
		out.Append(outputs.Textf("%v/",
			m.Position().Round(time.Second)).OnClick(nil))
	}
	out.Append(outputs.Textf("%v",
		m.Length.Round(time.Second).OnClick(nil)))
	return out
})
```

`.OnClick(nil)` in the above example prevents the default click handler of the media module from
being added to part of the output.

## Data: `type Info struct`

### Fields

* `PlaybackStatus PlaybackStatus`: Playing, Paused, Stopped, or Disconnected.
* `Shuffle bool`: Whether the media player is in shuffle mode.
* `Length time.Duration`: Length of the current track.
* `Title string`: Title of the current track.
* `Artist string`: Artist of the current track, can differ from AlbumArtist.
* `Album string`: Name of the Album the current track is from.
* `AlbumArtist string`: Album Artist as set on the Album.
* `ArtURL string`: URL to the artwork for the current track. (useful in, e.g. a notification).

### Methods

* `Paused() bool`: True if the media player is connected and paused.
* `Playing() bool`: True if the media player is connected and playing media.
* `Stopped() bool`: True if the media player is connected but stopped.
* `Connected() bool`: True if the media player is connected.
* `Position() time.Duration`: Returns the current track position, based on the last update from the media player.

*Note*: Position() is a computed property because there are no regular updates from the media player
while a track is playing. When a track begins playing, the media module tracks the rate (usually 1.0),
the current time, and the current position. This provides enough information to compute the position at
any point afterwards.

Because there are no regular updates while a track is playing, the media module forces its own updates
every second as long as the track is playing. So if your Output function uses Position, it should work fine.

### Controller Methods

In addition to the data methods listed above, media's `Info` type also provides controller
methods to interact with the player:

* `Play()`: Resumes playback of the current track.
* `Pause()`: Pauses the track. No effect if not playing.
* `PlayPause()`: Toggles between play and pause on the media player.
* `Stop()`: Stops (and usually clears) the currently playing track.
* `Next()`: Switches to the next track
* `Previous()`: Switches to the previous track (or restarts the current track, implementation is player-dependent).
* `Seek(time.Duration)`: Seeks to the specified offset from the current position. Negative durations seek backwards.
