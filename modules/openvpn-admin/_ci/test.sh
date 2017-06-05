#!/bin/bash
# Run the tests for this Go app in a Docker container so that all the OS user and config changes made by those tests
# don't mess up the host OS

set -e

readonly args="$@"

# Note that we manually start rsyslog, as Docker containers don't have a functioning init.d to automatically run it
# during boot.
docker-compose run --entrypoint "bash -c" openvpn-admin "cd src && rsyslogd && go test -v $args"