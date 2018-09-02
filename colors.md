---
title: Colours
---

Barista includes the ability to build a colour scheme for use in the bar, allowing consistent styles
across modules. It also provides methods to load colours schemes from various places:

## Accessing Colours

* `Scheme(string)`: returns the named colour from the scheme, or `nil` if not found.
* `Hex(string)`: returns the parsed hex colour, or `nil` if it failed to parse.

## Loading Colour Schemes

* `LoadBarConfig()`: loads the colors set in the current bar config. Some examples include
  - `statusline`: foreground colour
  - `background`: background colour
  - `separator`: colour of the separator line between segments
  - Workspaces:
	  - `focused_workspace_border`, `focused_workspace_bg`, `focused_workspace_text`
	  - `active_workspace_border`, `active_workspace_bg`, `active_workspace_text`
	  - `inactive_workspace_border`, `inactive_workspace_bg`, `inactive_workspace_text`
	  - `urgent_workspace_border`, `urgent_workspace_bg`, `urgent_workspace_text`
  - Binding mode: `binding_mode_border`, `binding_mode_bg`, `binding_mode_text`


* `LoadFromConfig(string)`: loads all "color_*" directives from an i3status.conf file, providing
  them without the color prefix. For example, if the i3status.conf file declares
  `color_good = '#07f'`, then loading it will make `colors.Scheme("good") = "#0077ff"`.

* `LoadFromArgs([]string)`: loads colors from command line arguments, to make it easier to load
  xresources values. i3 allows xresources values in its config file, using something like 
  `set_from_resource $black color0 #000000`. You can then provide that value to the bar through
  command line arguments, by setting the status_command to `bar black=$black`, and calling
  `LoadFromArgs(os.Args)` in the bar.

* `LoadFromMap(map[string]string)`: Load colors from a simple map of name -> hex strings. Useful for
  defining custom colours all in one location, allowing easy tweaking. 
