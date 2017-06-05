#!/usr/bin/env bash
#
# Script used by gruntwork-install to install the install-openvpn module.
#
set -e

# Locate the directory in which this script is located
readonly script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Move the bin files into /usr/local/bin
sudo cp "${script_path}"/bin/install-openvpn /usr/local/bin

# Change ownership and permissions
sudo chmod +x /usr/local/bin/install-openvpn

# Move the files in files/ to a dedicated directory for the gruntwork-installer
sudo mkdir -p /gruntwork/install-openvpn
sudo cp -R "${script_path}/files/." /gruntwork/install-openvpn/
sudo cp -R "${script_path}/scripts/." /gruntwork/install-openvpn/
sudo chmod +r /gruntwork/install-openvpn/*