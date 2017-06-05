#!/bin/bash

# This script is used by openvpn-admin to wrap the sourcing of the necessary variables (vars.local) and then
# to pass the call along to the ./build-key script. This is necessary because I could not get a working
# solution to sourcing the vars.local file directly in the Go exec.Command call.

source ./vars.local
KEY_NAME="" ./build-key --batch $1