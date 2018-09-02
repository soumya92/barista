---
title: testing/Cron
---

The `cron` testing package provides a way to run a test with retries, but only when running in
"cron" mode (usually triggered by the CI). This should be used for tests that depend on external
resources (especially network calls), where testing once a day just to ensure format compatibility
is sufficient.

Barista tests some weather providers and all icon fonts in cron mode daily.

## Example

```go
cron.Test(t, func() error {
	body, err := httpget("google.com")
	if err != nil {
		return err
	}
	require.Contains(t, string(body), "Google")
	return nil
})
```

## Failures and Retries

Cron runs the test function and retries a few times if it returns an error, in an attempt to avoid
transient failures (e.g. network problems). There is an increasing delay between retries.

However, failing the test will stop further retries. So the test function should be structured in a
way that errors from transient failures (e.g. network calls) are returned as-is, while errors that
cannot be fixed with a retry fail the test immediately.

In the above example, you can see that any errors from httpget are returned, causing a limited
number of retries, but if httpget succeeds and the body does not contain the string, the test fails
immediately, saving a few network requests and a lot of time.
