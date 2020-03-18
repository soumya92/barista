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
[![GoDoc](https://godoc.org/github.com/soumya92/barista?status.svg)](https://godoc.org/github.com/soumya92/barista)
[![Go Report Card](https://goreportcard.com/badge/github.com/soumya92/barista)](https://goreportcard.com/report/github.com/soumya92/barista)
[![Coverage Status](https://coveralls.io/repos/github/soumya92/barista/badge.svg?branch=master)](https://coveralls.io/github/soumya92/barista?branch=master)

Barista is an i3 status bar written in golang.

**This is not an official Google product**

To emphasize how not-Google this project is, the ability to use Google OAuth services (GSuite mail and calendar) may be removed from the default bar in the upcoming weeks. This is necessary to prevent the shared cloud project from being shutdown for violating Terms of Service (and to keep me from losing access to my email, movies, music, mobile apps, phone number, etc.).

**This only affects the pre-built binaries produced on CI and linked from GitHub. Any custom binaries are not affected.**

Users are free to build their own variant as long as they can provide the necessary API keys. It does seem that "personal use" is not subject to the same degree of scrutiny, only when the project sees more than ~50 users does it become problematic. So if each user creates a separate cloud project, this should never be a problem.

I have submitted the cloud project used by the CI for "review", but unless I receive confirmation that the project is in the clear, I will take this step as a precaution. If it's cleared later, I will reinstate GSuite mail and calendar support.

Locally built binaries using a different client ID will store their configuration in a different file, so you will need to re-authenticate once if you switch to a locally built binary. However, the previous configuration will remain available, in the event that I am able to provide sample-bar with GSuite modules in the future.

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
