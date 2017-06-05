#!/usr/bin/env bash
#
# Script used by gruntwork-install to install the process-certificate-requests module.
#

set -e

# Locate the directory in which this script is located
readonly script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Move the bin files into /usr/local/bin
sudo cp "${script_path}"/bin/run-process-requests /usr/local/bin
sudo cp "${script_path}"/bin/run-process-revokes /usr/local/bin

# Change ownership and permissions
sudo chmod +x /usr/local/bin/run-process-requests
sudo chmod +x /usr/local/bin/run-process-revokes
