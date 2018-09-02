---
title: OpenWeatherMap
---

Provides weather using the [OpenWeatherMap API](https://openweathermap.org/api).

## Usage

OpenWeatherMap allows several ways of specifying the location. See the
[OpenWeatherMap API documentation](https://openweathermap.org/current) for more details on location
specifiers.

* By City ID. Get City IDs from the OpenWeatherMap [city.list.json.gz](http://bulk.openweathermap.org/sample/)

  ```go
weather.New(openweathermap.CityID("2172797").Build())
```

* By City Name

  ```go
weather.New(openweathermap.CityName("Cairo", "Egypt").Build())
```

* From geographical coördinates, using the standard convention of north and west being negative
  values, and south and east being positive.

  ```go
weather.New(openweathermap.Coords(-34.3852, 132.4553).Build())
```

* From a postal code (or zip code)

  ```go
weather.New(openweathermap.CityName("SW1A 1AA", "UK").Build())
```

## Configuration

* `APIKey(string)`: Provide a different API key in case you run into quota issues with the shared one:
  
```go
weather.New(openweathermap.CityID("...").APIKey("apikeyhere").Build())
```
