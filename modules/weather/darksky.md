---
title: Dark Sky
---

Provides weather using the [Dark Sky API](https://darksky.net/dev). An API key is required.

## Usage

```go
weather.New(darksky.APIKey("...").Coords(lat, lon).Build())
```

where `lat`, `lon` are the latitude and longitude values. For longitude, negative values are west
and positive values are east. For latitude, negative values are north and positive values are south.
