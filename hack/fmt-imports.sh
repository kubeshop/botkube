#!/usr/bin/env bash
#
# This script formats import statements in all Go files.

# standard bash error handling
set -o nounset # treat unset variables as an error and exit immediately.
set -o errexit # exit immediately when a command fails.
set -E         # needs to be set if we want the ERR trap

CURRENT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT_DIR=$(cd "${CURRENT_DIR}/.." && pwd)

imports::install() {
	echo "Ensuring goimports-reviser installed..."
	go install github.com/mszostok/goimports-reviser/v2@f4a8f06bf75ef4e9c91d7039394f84ad379393bd
}

imports::format() {
	echo "Executing goimports-reviser..."
  pushd "$REPO_ROOT_DIR" > /dev/null

  paths=$(find . -name '*.go')

	for file in $paths; do
		goimports-reviser -file-path "$file" -rm-unused -local github.com/kubeshop/botkube -project-name github.com/kubeshop/botkube
  done

  popd > /dev/null
}

main() {
	imports::install

	imports::format
}

main
