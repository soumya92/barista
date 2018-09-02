---
title: testing/Bar
import_as: testBar
---

The `bar` testing package provides a test harness for modules, with methods to get the next output
from the hosted modules. Set up the test bar with `testBar.New(t)`, which also sets the `timing`
package into test mode as well. Then stream modules using `testBar.Run(module...)`.

Once streaming, there are a few methods that are useful:

- `AssertNoOutput()`: Asserts that there is no new output from any module (in 10ms of real time).
- `NextOutput()`: Returns output assertions against the next output (or fails the test if there
  was no output).
- `LatestOutput(...int)`: Returns output assertions against the latest output after making sure that
  each of modules provided as arguments has output at least once.
- `Tick()`: advances test time to the next scheduler and triggers it.
