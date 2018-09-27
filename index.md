<!-- untitled -->
# Quickstart

- Download the precompiled [sample-bar](https://github.com/soumya92/barista/blob/master/samples/sample-bar/sample-bar.go)
	somewhere memorable, and make it executable.

	```shell
curl -L https://git.io/fA7iT -o ~/bin/mybar
chmod +x ~/bin/mybar
```

- Set the status_command of the bar to this new binary

	`exec` is required for signal handling in some situations, and a pango font is required for
	[icon fonts](pango/icons) and inline [text formatting](pango).

	```
bar {
	status_command exec ~/bin/mybar
	font pango:DejaVu Sans Mono 10
}
```

- If you haven't previously setup oauth:

  ```shell
  ~/bin/mybar setup-oauth
  ```

  This will prompt you to go through the required oauth flow(s), and save **encrypted** tokens in
  `~/.config/barista` (or `$XDG_CONFIG_HOME/barista` if defined). The encryption key will be saved
  to the `login` keyring. See [the `go-keyring` README](https://github.com/zalando/go-keyring#linux) for more details.

- Restart i3

<div style="text-align: center">
	<img src="/assets/images/sample-bar-light-screenshot.png" alt="Another screenshot" height="22" />
	<br />
	<img src="/assets/images/sample-bar-screenshot.png" alt="Screenshot" height="22" />
</div>

If the icons are missing from the bar, you may need to [install the icon fonts](pango/icons#default-installation)
to `~/Github/`, or adjust the code if they are already available in a different location.

See [Customising Barista](/customising) for how to get started with building your own version, where
you can customise the output format, the modules and their order, and add arbitrary go code.

# Built-in Modules

Barista provides several modules out of the box:

- [battery](modules/battery): Shows battery information, with aggregation support.
- [clock](modules/clock): Shows the date/time, with timezone support.
- [counter](modules/counter): A simple module that demonstrates interactivity.
- [cpuload](modules/cpuload): Shows the load average.
- [cputemp](modules/cputemp): Shows the temperature of various components.
- [diskio](modules/diskio): Shows disk I/O rates.
- [diskspace](modules/diskspace): Shows available space for a disk.
- [funcs](modules/funcs): Provides modules that execute a given go function,
	either on click, or at a fixed interval.
- [github](modules/github): Shows unread notification count from GitHub, using oauth.
- [gmail](modules/gsuite/gmail): Shows unread thread count from Gmail, using oauth.
- [calendar](modules/gsuite/calendar): Shows events from Google Calendar, using oauth.
- [media](modules/media): Shows the currently playing track from an MPRIS player,
	and controls it using mouse events.
- [meminfo](modules/meminfo): Shows memory information.
- [netinfo](modules/netinfo): Shows network link information, such as connection state,
	hardware address, or IP addresses.
- [netspeed](modules/netspeed): Shows data transfer rates for a network interface.
- [reformat](modules/reformat): Wraps an existing module but changes the output format.
- [shell](modules/shell): Provides a module to show the output of a shell command (periodically or on-demand),
	and another one to show the last line of output from a long-running shell command.
- [sysinfo](modules/sysinfo): Shows system information.
- [volume](modules/volume): Shows the current alsa or pulseaudio volume,
	and allows controlling it using mouse events.
- [vpn](modules/vpn): Shows a simple connected/waiting/disconnected status for a tunnel interface.
- [weather](modules/weather): Shows the current weather conditions from a variety of providers.
- [wlan](modules/wlan): Shows network link information, augmented by wireless information
	such as the SSID, BSSID, channel, and frequency.

In addition to simple modules, barista also allows grouping several modules together and selectively
displaying their output on the bar. For example, [`group/switching`](group/switching) can be used to
show a single module from a group of many, and switch between them using buttons. See [`group`](group)
for more details.

# Formatting Output

Most modules allow specifying a custom function that receives some data related to the module, and
returns the output to be displayed on the bar. Some simple output functions are available in the
[`outputs`](outputs) package.

When using a pango font, i3bar also supports
[Pango Markup](https://developer.gnome.org/pango/stable/PangoMarkupFormat.html), which can be used
for rich in-line text formatting and icon fonts. Barista provides the [`pango`](pango) package for
constructing and manipulating output that uses pango markup.

# Custom Modules

Writing a custom barista module is fairly straightforward. Anything that can be done in a go program
can usually be adapted to display something on a barista bar.

If the module is polling something on an interval, it's easiest to write a go function and call it
using `funcs.Every()`. See [an example](modules/funcs#example-1) in the funcs package.

Or if you just need the output from a command, use the `shell` package. Some [examples](modules/shell#examples)
of shell commands being used for bar output are available in the package documentation.

For more details and a complete example, see the guide to [writing a module](docs/writing-a-module). For an example
of integrating a third-party go package, see the
[yubikey sample module](https://github.com/soumya92/barista/blob/master/samples/yubikey/yubikey.go).

