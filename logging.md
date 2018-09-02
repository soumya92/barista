---
title: Logging
import_as: l
---

Barista hides logging behind a compile time tag `debuglog`. When compiled without the tag, all log
functions are nops, which should permit elimination (or at least inlining) by the compiler, creating
a none to minimal performance impact.

When compiled with `-tags debuglog`, logging is enabled for all code that uses the barista logging
package. By default, logs are sent to stderr, which can require some creative redirection in order
to see them. However, you can also `SetOutput(io.Writer)` to send the logs to a file instead.

In addition to standard logs, which should only be used for important events or warnings/errors,
there are also fine logs, which can be enabled on a per-package basis using the `--finelog`
command line argument at runtime. The format is `--finelog=$pkg`, where pkg is a full package name
or a prefix, e.g. `--finelog=example.org/foo` will enable fine logging for packages under
`example.org/foo`, i.e. `example.org/foo` and `example.org/foo/bar`. There are also some special
package aliases for barista code, e.g. `mod:$module` for built-in modules.

To see all logs, use `--finelog=` (empty string), which matches all packages. This will print the
package string for each statement, which will help narrow down the list of packages that should be
given to the finelog argument on the command line.
