#!/bin/bash
# Build a Linux binary for ssh-iam. This script is most

set -e

readonly default_dest="$script_path/../bin/openvpn-admin"
readonly dest="${1-$default_dest}"

readonly script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly src="$script_path/../src"

echo "Compiling openvpn-admin Linux binary from $src into $dest"
cd "$src"
gox -os "linux" -arch "amd64" -output "$dest" -ldflags "-X main.VERSION=test-version"
