#!/bin/bash

# This script is used by openvpn-admin to wrap the sourcing of the necessary variables (vars.local) and then
# to pass the call along to the ./revoke-full script. This is necessary because I could not get a working
# solution to sourcing the vars.local file directly in the Go exec.Command call.

source ./vars.local
result=$(./revoke-full $1 2>&1 >/dev/null)

if [[ "$result" != *"Data Base Updated"* ]]; then
    echo $result
    exit 1
fi

exit 0