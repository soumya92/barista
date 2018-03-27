# Barista

[![Build Status](https://travis-ci.org/soumya92/barista.svg?branch=master)](https://travis-ci.org/soumya92/barista)
[![GoDoc](https://godoc.org/github.com/soumya92/barista?status.svg)](https://godoc.org/github.com/soumya92/barista)
[![Go Report Card](https://goreportcard.com/badge/github.com/soumya92/barista)](https://goreportcard.com/report/github.com/soumya92/barista)
[![codecov](https://codecov.io/gh/soumya92/barista/branch/master/graph/badge.svg)](https://codecov.io/gh/soumya92/barista)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fsoumya92%2Fbarista.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Fsoumya92%2Fbarista?ref=badge_shield)

Barista is an i3 status bar written in golang.

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
  status_command exec ~/bin/mybar
  font pango:DejaVu Sans Mono 10
}
```

See the [wiki](https://github.com/soumya92/barista/wiki) for more details


## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Fsoumya92%2Fbarista.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Fsoumya92%2Fbarista?ref=badge_large)