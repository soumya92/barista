---
title: Weather
---

Show the current weather conditions: `weather.New(someProvider)`.

Weather supports displaying the current conditions using a variety of pluggable providers, with the
ability to add custom providers fairly easily. Provider is just

```go
type Provider interface {
	GetWeather() (Weather, error)
}
```

A few providers are included out of the box:

* [OpenWeatherMap](/modules/weather/openweathermap): This one has an API key shared amongst all
  users, so it may become less reliable as more people start using it. However, there is still an
  option to specify a custom API key if you already have one.

* [Aviation Weather](/modules/weather/metar): Parses METAR weather conditions. No API key
  required, but it is limited to any stations served by [aviationweather.gov](https://aviationweather.gov/).

* [DarkSky](/modules/weather/darksky): "Hyperlocal forecast", requires an API key.

## Configuration

* `Output(func(Weather) bar.Output)`: Sets the output format.

* `RefreshInterval(time.Duration)`: Sets the interval to wait before fetching updated weather
  information. Defaults to 10 minutes.

## Example

<div class="module-example-out">25C, 5mph NNW</div>
Show the current temperature and wind information:

```go
weather.New(openweathermap.Zipcode("94045", "US").Build()).
	Output(func(w weather.Weather) bar.Output {
		return outputs.Textf("%.0fC, %.0fmph %s",
			w.Temperature.Celsius(),
			w.Wind.Speed.MilesPerHour(),
			w.Wind.Direction.Cardinal())
	})
```

## Data: `type Weather struct`

### Fields

* `Location string`: The location for which the data is being provided, especially useful if using autolocation.
* `Condition Condition`: Current weather conditions.
* `Description string`: A short, human-readable description of the current weather, e.g. "partly cloudy".
* `Temperature unit.Temperature`: Apparent temperature.
* `Humidity float64`: Relative humidity, 0.0 to 1.0.
* `Pressure unit.Pressure`: Atmospheric pressure.
* `Wind Wind`:
	* `Direction`
	* `unit.Speed`
* `CloudCover float64`: Cloud cover percentage, 0.0 to 1.0.
* `Sunrise time.Time`: Sunrise time for today.
* `Sunset time.Time`: Sunset time for today.
* `Updated time.Time`: Time the weather info was last updated. This may be different from the last time it was fetched.
* `Attribution string`: The service providing the weather information.

### Supporting Types

* `type Condition int`: A set of possible weather conditions. See the
  [Condition godoc](https://godoc.org/github.com/soumya92/barista/modules/weather#Condition) for the full list.
* `type Direction int`: Represents a compass direction in degrees:
	* `Deg() int`: 0 - 360.
	* `Cardinal() string`: Cardinal direction, e.g. "N", "SW", "ESE".

Documentation for [unit.Temperature](https://godoc.org/github.com/martinlindhe/unit#Temperature),
[unit.Pressure](https://godoc.org/github.com/martinlindhe/unit#Pressure), and
[unit.Speed](https://godoc.org/github.com/martinlindhe/unit#Speed)
