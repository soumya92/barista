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
weather.New(openweathermap.New("apikey").CityID("2172797"))
```

* By City Name

  ```go
weather.New(openweathermap.New("apikey").CityName("Cairo", "Egypt"))
```

* From geographical coördinates, using the standard convention of south and west being negative
  values, and north and east being positive.

  ```go
weather.New(openweathermap.New("apikey").Coords(-34.3852, 132.4553))
```

* From a postal code (or zip code)

  ```go
weather.New(openweathermap.New("apikey").Zipcode("SW1A 1AA", "UK"))
```
