#!/usr/bin/env bash
#
# This script formats import statements in all Go files.

# standard bash error handling
set -o nounset # treat unset variables as an error and exit immediately.
set -o errexit # exit immediately when a command fails.
set -E         # needs to be set if we want the ERR trap

CURRENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT_DIR=$(cd "${CURRENT_DIR}/.." && pwd)
GOIMPORTS_REVISER_VERSION=3.3.1

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

imports::format() {
  echo "Executing goimports-reviser..."
  pushd "$REPO_ROOT_DIR" > /dev/null

  echo "- Revising cmd..."
  goimports-reviser -rm-unused -project-name github.com/kubeshop/botkube -recursive ./cmd
  echo "- Revising internal..."
  goimports-reviser -rm-unused -project-name github.com/kubeshop/botkube -recursive ./internal
  echo "- Revising pkg..."
  goimports-reviser -rm-unused -project-name github.com/kubeshop/botkube -recursive ./pkg
  echo "- Revising test..."
  goimports-reviser -rm-unused -project-name github.com/kubeshop/botkube -recursive ./test

  popd > /dev/null
}

main() {
  host::install::imports

  imports::format
}

main
