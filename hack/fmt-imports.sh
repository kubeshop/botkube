#!/usr/bin/env bash
#
# This script formats import statements in all Go files.

# standard bash error handling
set -o nounset # treat unset variables as an error and exit immediately.
set -o errexit # exit immediately when a command fails.
set -E         # needs to be set if we want the ERR trap

CURRENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT_DIR=$(cd "${CURRENT_DIR}/.." && pwd)
GOIMPORTS_REVISER_VERSION=2.5.3

host::os() {
  local host_os
  case "$(uname -s)" in
    Darwin)
      host_os=darwin
      ;;
    Linux)
      host_os=linux
      ;;
    *)
      echo "Unsupported host OS. Must be Linux or Mac OS X."
      exit 1
      ;;
  esac
  echo "${host_os}"
}

host::arch() {
  local host_arch
  case "$(uname -m)" in
    x86_64*)
      host_arch=amd64
      ;;
    i?86_64*)
      host_arch=amd64
      ;;
    amd64*)
      host_arch=amd64
      ;;
    aarch64*)
      host_arch=arm64
      ;;
    arm64*)
      host_arch=arm64
      ;;
    arm*)
      host_arch=arm
      ;;
    ppc64le*)
      host_arch=ppc64le
      ;;
    *)
      echo "Unsupported host arch. Must be x86_64, arm, arm64, or ppc64le."
      exit 1
      ;;
  esac
  echo "${host_arch}"
}

host::install::imports() {
  readonly INSTALL_DIR="${REPO_ROOT_DIR}/bin"
  mkdir -p "${INSTALL_DIR}"
  export PATH="${INSTALL_DIR}:${PATH}"

  echo "Install goimports-reviser ${GOIMPORTS_REVISER_VERSION} to local ./bin..."

  os=$(host::os)
  arch=$(host::arch)
  name="goimports-reviser_${GOIMPORTS_REVISER_VERSION}_${os}_${arch}"

  pushd "${INSTALL_DIR}"

  # download the release
  curl -L -O "https://github.com/incu6us/goimports-reviser/releases/download/v${GOIMPORTS_REVISER_VERSION}/${name}.tar.gz"

  # extract the archive
  tar -zxvf "${name}".tar.gz

  popd
}

imports::files_to_check() {
  pushd "$REPO_ROOT_DIR" > /dev/null
  paths=$(git diff --name-only main  '**/*.go')
  popd > /dev/null

  echo "$paths"
}
imports::format() {
  paths=$1

  echo "Executing goimports-reviser..."
  pushd "$REPO_ROOT_DIR" > /dev/null

  # TODO: Consider to run it in parallel to speed up the execution.
  for file in $paths; do
    echo "Formatting $file..."
    goimports-reviser -file-path "$file" -rm-unused -local github.com/kubeshop/botkube -project-name github.com/kubeshop/botkube
  done

  popd > /dev/null
}

main() {
  filesToCheck=$(imports::files_to_check)
  if [ -z "$filesToCheck" ]
  then
    echo "Skipping executions as no files were modified."
    exit 0
  fi

  host::install::imports
  imports::format "$filesToCheck"
}

main
