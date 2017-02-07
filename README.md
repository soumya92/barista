# Bar-i-sta

Bar-i-sta is an i3 status bar written in golang.

**This is not an official Google product**

## Features

- Based on push rather than fixed interval polling. Currently only media and
  volume benefit from this, but this opens the door to async updates from
  shell scripts, irc, &c.

- Produces a single binary via go build. This makes it easy to set up the bar
  executable, since no import paths, environment variables, et al. need to be
  configured.

- Good click handlers (especially media and volume), since we can wait for a
  command and update the bar immediately rather than waiting for the next 'tick'.

- Configuration is code, providing oodles of customization options without
  needing myriad configuration options in a file somewhere. If/then/else, loops,
  functions, variables, and even other go packages can all be used seamlessly.

## Usage

See samples/sample-bar.go for a sample bar.

To build your own bar, simply create a `package main` go file,
import and configure the modules you wish to use, and call `barista/bar.Run(...)`.

To show your bar in i3, set the `status_command` of a `bar { ... }` section
to be the newly built bar binary, e.g.

```
bar {
  position top
  status_command ~/bin/mybar
}
```
