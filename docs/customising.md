---
pkg: none
title: Customising Barista
---

Building your own bar requires a few additional steps, because we cannot distribute API keys or
client secrets in an open source project.

The simplest way to build a custom bar is to start with
[`simple.go`](https://github.com/soumya92/barista/blob/master/samples/simple/simple.go)
as a template, and experiment with changes to the file.

<div style="text-align: center">
  <img src="/assets/images/simple-light-screenshot.png" alt="Screenshot of simple bar" height="22" />
  <br />
  <img src="/assets/images/simple-screenshot.png" alt="Another screenshot of simple bar" height="22" />
</div>

  - Setup the `barista.run` package in `$GOPATH`:

  ```shell
go get barista.run
```

  - Download the sources:

  ```shell
curl -L https://git.io/JUq6x -o mybar.go
```

  - To make it easier to work with the placeholders in this file, download the build script:
  
  ```shell
curl -L https://git.io/JUqiv -o build.sh
chmod +x build.sh
```

  - Edit `build.sh` and change `TARGET_FILE` to point to the go file with the placeholders.
    For example,
  
  ```bash
TARGET_FILE="mybar.go"
```

  - Set up the necessary OAuth and API keys in `~/.config/barista/keys`, following the format:

  ```
OWM_API_KEY="..."
GITHUB_CLIENT_ID="..."
GITHUB_CLIENT_SECRET="..."
```

  - Run `./build.sh`, with any arguments that you would pass to `go build`, e.g.
  
  ```shell
./build.sh -o ~/bin/mybar -i mybar.go
```

  - Restart i3 to see the changes

  ```shell
i3-msg restart
```

The rest of the steps, for saving oauth tokens and setting up icon fonts, are the same as the
[quickstart](/#quickstart).

## Caching HTTP Requests for Development

When customising the bar, you may need to rebuild and restart the bar a lot, and each restart will
issue HTTP requests to external services (weather, calendar, etc.). To avoid consuming excessive
quota during development, it may be desirable to use the [`httpcache`](/testing/httpcache) package.

See the package documentation for a simple example of setting up global HTTP caching for the binary.
Remember to remove the caching once you're satisfied with your barista setup.

# OAuth and Client Keys

Quick links to the sign-up pages for the various available services:

- [OpenWeatherMap](https://openweathermap.org/appid)
- [GitHub](https://github.com/settings/applications/new)
- [Google](https://developers.google.com/identity/protocols/OAuth2), follow the link to the
  [Google API Console](https://console.developers.google.com/).
