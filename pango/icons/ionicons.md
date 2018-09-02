---
title: Ionicons
---

> Beautifully crafted open source icons

* Prefixes: `ion-`, `ion-md-`, and `ion-ios-`.

* Repository: [`https://github.com/ionic-team/ionicons`](https://github.com/ionic-team/ionicons)

* Font file: `/docs/fonts/ionicons.ttf`

* Icon search: [Ionicons website](https://ionicons.com/)

The standard prefix is `ion-`, with an additional segment for the two styles, so
`pango.Icon("ion-md-cloud-download")` will return the "cloud-download" icon in Material style.

Ionicons provides icons in two styles: Material and iOS. By default, each style is available with
an additional segment in the prefix, `ion-md-$icon` for Material style, and `icon-ios-$icon` for
iOS style. However you can choose to load a style by default, eliminating the additional segment.

```go
ionicons.LoadIos("/path/to/ionicons/repo")

pango.Icon("ion-color-filter") // loads the iOS style "color-filter" icon,
pango.Icon("ion-md-color-filter") // loads the Material style icon.
```

Similarly, `ionicons.LoadMd("...")` will load the Material style as the default, with iOS style
available behind the `ion-ios-` prefix.
