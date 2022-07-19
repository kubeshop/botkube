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
  go install github.com/incu6us/goimports-reviser/v2@latest
}

imports::format() {
  echo "Executing goimports-reviser..."
  pushd "$REPO_ROOT_DIR" > /dev/null

  paths=$(find . -name '*.go')

  # TODO: Consider to run it in parallel to speed up the execution.
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
