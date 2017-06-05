#!/bin/bash
# Build a binary for openvpn-admin that can be used in the openvpn-server-ubuntu1604.json Packer template.
#
# Note that we build the binary inside of a Docker container to ensure we are compiling on Linux. If we compiled on OS
# X instead and ran the code on Linux (e.g. on an Ubuntu AMI during testing), then the Go os/user package will not work
# correctly (e.g. user.Current() will return an error). For more info, see: https://github.com/golang/go/issues/6376

set -e

function remove_build_container {
  local readonly container_name="$1"

  # We want to clean up the container whether or not it exists, so we append "|| true" at the end to make sure this
  # whole program doesn't exist if docker rm exits with an error
  echo "Removing container $container_name"
  docker rm "$container_name" > /dev/null 2>&1 || true
}

function build_binary_in_docker_container {
  local readonly src_dir_host="$1"
  local readonly dest_path_on_container="$2"
  local readonly container_name="$3"

  echo "Building openvpn-admin binary in Docker container"
  cd "$src_dir_host"
  docker-compose run --entrypoint "bash -c" --name "$container_name" openvpn-admin "./scripts/build-linux-binary.sh $dest_path_on_container"
}

function copy_binary_to_host {
  local readonly src_path_on_container="$1"
  local readonly dest_dir_on_host="$2"
  local readonly container_name="$3"

  echo "Copying openvpn-admin binary from $src_path_on_container on Docker container $container_name to $dest_dir_on_host on host"
  mkdir -p "$dest_dir_on_host"
  docker cp "$container_name:$src_path_on_container" "$dest_dir_on_host"
}

function build_binary {
  local readonly script_path="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  local readonly container_name="openvpn-admin-linux-build"
  local readonly bin_path_on_container="/go/bin/openvpn-admin"

  remove_build_container "$container_name"
  build_binary_in_docker_container "$script_path/../../modules/openvpn-admin/" "$bin_path_on_container" "$container_name"
  copy_binary_to_host "$bin_path_on_container" "$script_path/../bin/" "$container_name"
  remove_build_container "$container_name"
}

build_binary




