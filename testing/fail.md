---
title: testing/Fail
---

The `fail` testing package provides meta-assertions: assertions about other assertions. It can be
used to verify that test helpers correctly pass or fail the test. It runs test functions with a
fake testing.T, and asserts against the result.

## Examples

A simple assertion:
```go
fail.AssertFails(func(t *testing.T) {
	require.Equal(t, 5, 2+2)
})
```

Sometimes there is some setup involved, and you want to make sure the test failure is from the
failing assertion, and not the setup code. In that case, use `Setup`:

```go
var thing TestThing
fail.Setup(func(t *testing.T){
	thing = makeTestThing(t)
}).AssertFails(func(*testing.T) {
	thing.AssertTrue(false)
})
```

In this example, the test will fail if `makeTestThing` fails the test, and it will also fail if
`AssertTrue(false)` does *not* fail the test.
