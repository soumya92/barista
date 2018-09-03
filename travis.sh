#!/usr/bin/env bash

CC_TEST_REPORTER_LOC="https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64"

set -e
mkdir -p profiles/

CODECLIMATE=0
if test -n "$CC_TEST_REPORTER_ID" && curl -LSs "$CC_TEST_REPORTER_LOC" >./cc-test-reporter; then
	CODECLIMATE=1
	chmod +x ./cc-test-reporter
	./cc-test-reporter before-build
fi

# Run tests with coverage for all barista packages
go list ./... \
| grep -v /samples/ \
| tac \
| xargs -n1 -P4 -IPKG sh -c \
'go test -coverprofile=profiles/$(echo "PKG" | sed "s|/|_|g").out -race -covermode=atomic "PKG"'

# Debug log tests need the build tag, otherwise the nop versions will be used.
go test -tags debuglog -coverprofile=profiles/logging_real.out -race -covermode=atomic ./logging

# Merge all code coverage reports.
cat profiles/*.out > coverage.txt
rm -rf profiles/

# Run tests only for samples.
# This is just to make sure that all samples compile.
go test ./samples/...

# Codecov.io wants coverage.txt, but CodeClimate wants c.out.
if [ $CODECLIMATE -eq 1 ]; then
	echo "mode: count" > c.out
	grep -h -v "^mode:" coverage.txt >> c.out
	./cc-test-reporter after-build
fi
