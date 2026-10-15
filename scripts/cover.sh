#!/bin/sh
# Print statement coverage for the module and write a profile for the CML parser.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$root"
profile=${COVERPROFILE:-coverage.out}

echo "== packages =="
go test ./... -count=1 -cover

echo
echo "== cml functions =="
go test ./cml -count=1 -coverprofile="$profile" -covermode=atomic
go tool cover -func="$profile"

echo
echo "profile: $profile"
echo "html:    go tool cover -html=$profile"
