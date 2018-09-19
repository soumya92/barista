---
title: Privacy Policy
pkg: none
---

Barista collects system and user information in order to display the configured
status modules. A limited subset of this information is stored on your local
disk (oauth tokens), but efforts have been made to ensure it is stored securely.

No information is transmitted to the Barista authors. Some information is transmitted
to third-parties to fulfil a limited subset of requests. These include:

- [freegeoip.app](https://freegeoip.app/) to determine location for weather.

  This can be disabled by providing a location manually.

- [OpenWeatherMap](https://openweathermap.org/privacy-policy) to get the weather.
  
  This can be disabled by using a different weather provider, but their privacy
  policy will apply. This can also be disabled by removing the weather module.

- [GitHub](https://help.github.com/articles/github-privacy-statement/) to display
  GitHub notifications.

  This can be disabled by removing the GitHub notifications module.

- [Google](https://policies.google.com/privacy) to display unread Gmail count and
  upcoming Calendar events.

  This can be disabled by removing the gmail and calendar modules.

These are all enabled by default in the sample-bar. You can choose to build your
own bar with a subset of the functionality, which will also restrict requests to
third-parties.

The barista website uses Google Analytics to track the most popular pages, in an
effort to better focus development efforts. This can be disabled by installing
the [Google Analytics opt-out browser add-on](https://support.google.com/analytics/answer/181881?hl=en).
