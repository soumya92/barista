---
title: GitHubFS
---

GitHubFS provides an [`afero` Filesystem](https://godoc.org/github.com/spf13/afero#Fs) that proxies
file system calls to GitHub. This allows reading a few files from a GitHub repository without
needing to clone it locally.

The paths are of the form `$user/$repo/$branch/$filename`. For example, reading the file
`/spf13/afero/master/afero.go` will return the contents of
[spf13/afero/afero.go](https://github.com/spf13/afero/blob/master/afero.go) from the master branch.

Create a new filesystem instance with `githubfs.New()`. There are no options to configure.
