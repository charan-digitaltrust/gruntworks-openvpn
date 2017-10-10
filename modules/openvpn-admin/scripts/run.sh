#!/bin/bash
# A simple script to run this Go app in dev mode

set -e

readonly script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$script_path/../src/"
go run $(ls *.go | grep -v _test.go) "$@"
