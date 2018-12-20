---
title: Dark Sky
---

Provides weather using the [Dark Sky API](https://darksky.net/dev).

## Usage

```go
weather.New(darksky.New("apikey").Coords(lat, lon))
```

where `lat`, `lon` are the latitude and longitude values. For longitude, negative values are west
and positive values are east. For latitude, positive values are north and negative values are south.
