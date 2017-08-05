#!/usr/bin/env bash

# From https://github.com/codecov/example-go#caveat-multiple-files
set -e
echo "" > coverage.txt

# Run tests with coverage for all barista packages
for d in $(go list ./... | grep -v barista/samples); do
	go test -coverprofile=profile.out -covermode=count $d
	if [ -f profile.out ]; then
		cat profile.out >> coverage.txt
		rm profile.out
	fi
done

# Run tests only for samples.
# This is just to make sure that all samples compile.
go test ./samples/...
