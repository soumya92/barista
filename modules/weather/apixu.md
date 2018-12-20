---
title: APIXU
---

Provides weather using the [APIXU API](https://www.apixu.com/api.aspx).

## Usage

```go
weather.New(APIXU.New("apikey").Query("query"))
```

The query string is in one of the [supported formats](https://www.apixu.com/doc/request.aspx):

- Latitude and Longitude (Decimal degree) e.g: `"48.8567,2.3508"`
- city name e.g.: `"Paris"`
- US zip e.g.: `"10001"`
- UK postcode e.g: `"SW1"`
- Canada postal code e.g: `"G2J"`
- metar:&lt;metar code&gt; e.g: `"metar:EGLL"`
- iata:&lt;3 digit airport code&gt; e.g: `"iata:DXB"`
- auto:ip IP lookup e.g: `"auto:ip"`
- IP address (IPv4 and IPv6 supported) e.g: `"100.0.0.1"`
