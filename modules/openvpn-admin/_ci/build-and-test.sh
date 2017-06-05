#!/bin/bash
# Build the Docker container and run the tests for this Go app in that Docker container so that all the OS user and
# config changes made by those tests don't mess up the host OS

set -e

docker build -t gruntwork/openvpn-admin .
./_ci/test.sh