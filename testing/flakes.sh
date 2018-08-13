#!/bin/sh
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

# This script runs the given test many times, in parallel, with all logging
# enabled, and prints the names of the files containing output from failures.
#
# This is useful for tracking down flaky tests, hence the name.

# To avoid depending on a second script, use a 'magic' first argument.
if [ "$1" = "BARISTA_FLAKE_TEST" ]; then
# When invoked by xargs using the magic first arg, it simply executes the
# given command, piping the stdout and stderr to a temporary file.
# If the command exits successfully (or ^C), nothing happens, otherwise
# the name of the file is printed.
	shift
	tmpdir="$1"
	shift
	outfile="$(mktemp --tmpdir="$tmpdir")"
	"$@" >"$outfile" 2>&1
	status="$?"
	if [ "$status" -eq 0 ] || [ "$status" -eq 127 ]; then
		rm -f "$outfile"
	else
		echo "$outfile"
	fi
	exit $status
fi

# When invoked normally, sets up a parallel pipeline using seq | xargs to
# run the given test many times and catching any failures.

# TODO: Use argument parsing here.
parallel=16
total=96
tmpdir="$(mktemp -d)"

echo "Saving results to $tmpdir"
seq 1 $total | xargs -n 1 -P $parallel $0 BARISTA_FLAKE_TEST "$tmpdir" go test -v -race -tags debuglog "$@" -- -finelog=
