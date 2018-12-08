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

# Since quite a few tests have sleeps, running nproc + 2 tests should result in most
# effective parallelisation.
NPAR="$(($(nproc) + 2))"

# For local runs, use golint from PATH,
GOLINT="$(which golint)"
# but fallback to the CI path otherwise.
[ -n "$GOLINT" ] || GOLINT="$HOME/gopath/bin/golint"

echo "Lint: $GOLINT ./..."
$GOLINT ./...

echo "Vet: go vet"
go vet

echo "Test: Running $NPAR in parallel"
# Run tests with coverage for all barista packages
go list ./... \
| grep -v /samples/ \
| sed "s|_$PWD|.|" \
| tac \
| xargs -n1 -P$NPAR -IPKG sh -c \
'go test -timeout 90s -coverprofile=profiles/$(echo "PKG" | sed -e "s|./||" -e "s|/|_|g").out -race -covermode=atomic "PKG"'

echo "Test: Logging with -tags debuglog"
# Debug log tests need the build tag, otherwise the nop versions will be used.
go test -tags debuglog -coverprofile=profiles/logging_real.out -race -covermode=atomic barista.run/logging

# Remove all _capi.go coverage since those will intentionally not be tested.
for profile in profiles/*.out; do
	perl -i -ne 'print unless /_capi\.go:/' "$profile"
done

# Merge all code coverage reports. Although codecov does something similar internally,
# doing it here means that after running ./test.sh, you can run
#     go tool cover -html=coverage.txt
# and it will show a coverage report instead of complaining about a bad format.
grep -E '^mode: \w+$' "$(find profiles/ -name '*.out' -print -quit)" > coverage.txt
grep -hEv '^(mode: \w+)?$' profiles/*.out >> coverage.txt
rm -rf profiles/

echo "Test: Samples"
# Run tests only for samples.
# This is just to make sure that all samples compile.
go test ./samples/...

# Codecov.io wants coverage.txt, but CodeClimate wants c.out.
if [ $CODECLIMATE -eq 1 ]; then
	cp "coverage.txt" "c.out"
	./cc-test-reporter after-build
fi
