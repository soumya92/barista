#!/usr/bin/env bash

# Copyright 2018 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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

# Merge all code coverage reports. Doing this here means that after running
# ./test.sh,
#     go tool cover -html=c.out
# will show a coverage report instead of complaining about a bad format.
grep -E '^mode: \w+$' "$(find profiles/ -name '*.out' -print -quit)" > c.out
grep -hEv '^(mode: \w+)?$' profiles/*.out >> c.out
rm -rf profiles/

echo "Test: Samples"
# Run tests only for samples.
# This is just to make sure that all samples compile.
go test ./samples/...

if [ $CODECLIMATE -eq 1 ]; then
	./cc-test-reporter after-build
fi
