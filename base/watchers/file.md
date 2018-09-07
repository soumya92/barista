---
title: watchers/File
---

The `file` package provides the ability to watch for changes to a single file, in a way that works
across deletions of the tree and handles non-existence of the file or part of the path to the file.
It also uses [coalescing notifications](/base/notifier), so extra processing is minimised.

Create a watcher: `w := file.Watch("/var/run/something.pid")`, and remember to unsubscribe using
`w.Unsubscribe()`.

Then simply listen for updates to the `Updates` channel, and errors on the `Errors` channel, e.g.
```go
for {
	select {
		case <-w.Updates:
			// Something happened, query file and update module output.
		case e := <-w.Errors:
			sink.Error(e)
			return  // No need to unsubscribe on errors.
	}
}
```
