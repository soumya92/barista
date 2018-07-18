#!/usr/bin/env bash

# From https://github.com/codecov/example-go#caveat-multiple-files
set -e
mkdir -p profiles/

# Run tests with coverage for all barista packages
go list ./... \
| grep -v barista/samples \
| tac \
| xargs -n1 -P4 -IPKG sh -c \
'go test -coverprofile=profiles/$(echo "PKG" | sed "s|/|_|g").out -race -covermode=atomic "PKG"'

# Debug log tests need the build tag, otherwise the nop versions will be used.
go test -v -tags debuglog -coverprofile=profiles/logging_real.out -race -covermode=atomic ./logging

# Merge all code coverage reports.
cat profiles/*.out > coverage.txt
rm -rf profiles/

# Run tests only for samples.
# This is just to make sure that all samples compile.
go test ./samples/...
