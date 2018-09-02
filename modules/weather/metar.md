---
title: METAR Aviation Weather
---

Provides weather using the METAR API from [NOAA Aviation Digital Data Service](https://www.aviationweather.gov/).

## Usage

```go
weather.New(metar.Station("KBFI").Build())
```

## Configuration

* `StripRemarks()`: Removes remarks from the description in the returned weather.
* `IncludeFlightCat()`: Adds `[$flight category] `&nbsp;to the beginning of the description.

```go
weather.New(metar.Station("...").StripRemarks().IncludeFlightCat().Build())
```
