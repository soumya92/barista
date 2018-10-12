---
pkg: none
title: Customising Barista
---

Building your own bar requires a few additional steps, because we cannot distribute API keys or
client secrets in an open source project.

The simplest way to build a custom bar is to start with
[`sample-bar.go`](https://github.com/soumya92/barista/blob/master/samples/sample-bar/sample-bar.go)
as a template, and experiment with changes to the file.

  - Download the sources:

  ```shell
curl -L https://git.io/fA4qJ -o mybar.go
```

  - To make it easier to work with the placeholders in this file, download the build script:
  
  ```shell
curl -L https://git.io/fA7iZ -o build.sh
chmod +x build.sh
```

  - Edit `build.sh` and change `TARGET_FILE` and `TARGET_PACKAGE` to match your configuration.
    `TARGET_FILE` should point to the go file with the placeholders, while `TARGET_PACKAGE` should
    point to the package (or single file) to build. For example,
  
  ```bash
TARGET_FILE="mybar.go"
TARGET_PACKAGE="mybar.go"
```

  - Set up the necessary OAuth and API keys in `~/.config/barista/keys`, following the format:

  ```
OWM_API_KEY="..."
GITHUB_CLIENT_ID="..."
GITHUB_CLIENT_SECRET="..."
```

  - Run `./build.sh`, optionally with any arguments that you would pass to `go build`, e.g.
  
  ```shell
./build.sh -o ~/bin/mybar
```

  - Restart i3 to see the changes

  ```shell
i3-msg restart
```

The rest of the steps, for saving oauth tokens and setting up icon fonts, are the same as the
[quickstart](/#quickstart).

# OAuth and Client Keys

Quick links to the sign-up pages for the various available services:

- [OpenWeatherMap](https://openweathermap.org/appid)
- [GitHub](https://github.com/settings/applications/new)
- [Google](https://developers.google.com/identity/protocols/OAuth2), follow the link to the
  [Google API Console](https://console.developers.google.com/).
