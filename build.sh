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

set -e

# Save keys to ~/.config/barista/keys to avoid leaking them to git.
# Or use your preferred way to set them as environment variables.
CONFIG_DIR="${XDG_CONFIG_HOME:-"$HOME/.config"}"
[ -e "$CONFIG_DIR/barista/keys" ] && . "$CONFIG_DIR/barista/keys"

# Any KEYS will be inserted into the sample bar by replacing
# '%%$NAME_OF_KEY%%' with the value of $NAME_OF_KEY from the environment.
# e.g. '%%OWM_API_KEY%%' in the go file will be replaced with the value
# of $OWM_API_KEY.
KEYS=(
	'GITHUB_CLIENT_ID' 'GITHUB_CLIENT_SECRET'
	'GOOGLE_CLIENT_ID' 'GOOGLE_CLIENT_SECRET'
	'OWM_API_KEY'
)

TARGET_FILE="./samples/simple/simple.go"

# Save the current sample-bar, so we can revert it after building, to
# prevent accidentally checking in the keys. We can't use git checkout
# because the file could have other modifications that aren't committed.
BACKUP_FILE="$(mktemp)"
cp "$TARGET_FILE" "$BACKUP_FILE"
function restore {
	cp "$BACKUP_FILE" "$TARGET_FILE"
	rm -f "$BACKUP_FILE"
}
trap restore EXIT

for KEY in ${KEYS[@]}; do
	if [ -n "${!KEY}" ]; then
		sed -i "s/%%${KEY}%%/${!KEY}/g" "$TARGET_FILE"
	else
		echo "Skipping $KEY, value not set" >&2
	fi
done

# Build the sample bar with all the keys set. Pass all arguments to the
# `go build` command, allowing e.g. `./build.sh -o ~/bin/mybar`, or even
# `./build.sh -race -tags debuglog`.
go build "$@"

