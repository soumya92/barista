---
title: testing/Module
import_as: testModule
---

The `module` testing package provides a `TestModule` that can be used to test the behaviour of
module hosts, by providing methods to control the output, and make assertions against click events.

Create a new test module using `testModule.New(t)`. Once created, it can be used wherever a
bar.Module is required.

It provides one configuration option: `SkipClickHandlers()`. If called, the module will not add
click handlers to its output. This will create outputs that can be compared, but will prevent
click assertions from working.

Once streaming, a test module provides a few methods to trigger certain behaviour:

- `Output(bar.Output)`/`OutputText(string)`: Send a new output to the sink currently streaming to.
- `Close()`: Causes the Stream() function to return.

It also provides some assertion methods:

- `AssertStarted()`: Waits for the Stream function to be called. Fails the test if it's not called.
- `AssertNotStarted()`: Asserts that the Stream function has not been called. Only valid before the
  first Stream() or after Close(). This will fail the test if the module is currently streaming.
- `AssertClicked()`: Returns the bar.Event that triggered the default OnClick handler. Fails the
  test if no segment has been clicked.
- `AssertNotClicked()`: Asserts that none of the default OnClick handlers were triggered.
