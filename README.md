<!--
Copyright 2018 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->
<img src="https://raw.githubusercontent.com/soumya92/barista/gh-pages/logo/128.png" height="128" width="128" alt="Logo" />

# Barista

[![Release](https://github.com/soumya92/barista/workflows/Release/badge.svg)](https://github.com/soumya92/barista/releases/tag/autorelease)
[![GoDoc](https://godoc.org/barista.run?status.svg)](https://godoc.org/barista.run)
[![Maintainability](https://api.codeclimate.com/v1/badges/753f34fc34df1a05b05f/maintainability)](https://codeclimate.com/github/soumya92/barista/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/753f34fc34df1a05b05f/test_coverage)](https://codeclimate.com/github/soumya92/barista/test_coverage)

Barista is an i3 status bar written in golang.

**This is not an official Google product**

## Features

- Based on push rather than fixed interval polling. This allows immediate updates
  for many modules, like volume, media, shell, etc.

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
import and configure the modules you wish to use, and call `barista.Run()`.

To show your bar in i3, set the `status_command` of a `bar { ... }` section
to be the newly built bar binary, e.g.

```
bar {
  position top
  status_command exec ~/bin/mybar
  font pango:DejaVu Sans Mono 10
}
```

See the [quickstart](https://barista.run/#quickstart) for more details.
