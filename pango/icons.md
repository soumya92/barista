---
title: Icon Fonts
pkg: noimport
---

Although i3bar can only display text, it supports [Pango Markup](https://docs.gtk.org/Pango/pango_markup.html),
which means that any icon font installed on the system can be used to add pictograms to the bar.

There are a few simple steps to using an icon font in your bar:

- Clone the icon font's repository
  
  This gets you the font files as well as the list of icon names and code points.

- Symlink the font
  
  ```shell
  ln -s repo/path/to/font.ttf ~/.fonts/
  ```
  
  Symlinking ensures that the list of icons stays in sync with the font loaded by the
  system. Some icon fonts do not keep their code points the same between updates.

- Load the mapping in your bar
  
  ```go
  fontprovider.Load("/path/to/repo")
  ```
  
  The provider will use an appropriate file relative to the repository path to determine
  which icons are supported and what code points render each icon. Once loaded, 
  `pango.Icon("provider-iconname")` will give you a pango Node that has the appropriate
  font face (and any other required attributes) set, and can be used in an output function.


## Supported Fonts

- [Material Design Icons](/pango/icons/material)
  
  Delightful, beautifully crafted symbols for common actions and items  
  `pango.Icon("material-settings-ethernet")`

	- [Community-led Iconography](/pango/icons/mdi)

	  2700+ Material Design Icons from the Community  
	  `pango.Icon("mdi-music-circle")`

- [FontAwesome Free](/pango/icons/fontawesome)
  
  "the web’s most popular icon set and toolkit"  
  `pango.Icon("fa-comment-dots")`

- [Typicons](/pango/icons/typicons)
  
  A Free Icon Font by Stephen Hutchings  
  `pango.Icon("typecn-flag-outline")`

## Default Installation

The default sample bar assumes that icon fonts have been cloned to `~/Github/`,
and are installed via symlink (or directly) in a font folder (usually `~/.fonts/`).
To install all supported icon fonts, simply run the commands below from your home directory.

```shell
mkdir Github
cd Github

# Material Design Icons
git clone --depth 1 https://github.com/google/material-design-icons
ln -s $PWD/material-design-icons/font/MaterialIcons-Regular.ttf ~/.fonts/

# Community Fork
git clone --depth 1 https://github.com/Templarian/MaterialDesign-Webfont
ln -s $PWD/MaterialDesign-Webfont/fonts/materialdesignicons-webfont.ttf ~/.fonts/

# FontAwesome
git clone --depth 1 https://github.com/FortAwesome/Font-Awesome
ln -s "$PWD/Font-Awesome/otfs/Font Awesome 5 Free-Solid-900.otf" ~/.fonts/
ln -s "$PWD/Font-Awesome/otfs/Font Awesome 5 Free-Regular-400.otf" ~/.fonts/
ln -s "$PWD/Font-Awesome/otfs/Font Awesome 5 Brands-Regular-400.otf" ~/.fonts/

# Typicons
git clone --depth 1 https://github.com/stephenhutchings/typicons.font
ln -s $PWD/typicons.font/src/font/typicons.ttf ~/.fonts/
```

You may need to rebuild the font cache using `fc-cache -fv`, and restart i3 to pick up the new fonts.
